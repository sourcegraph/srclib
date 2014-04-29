package srcgraph

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"strings"

	"github.com/aybabtme/color/brush"
	"github.com/sourcegraph/go-vcs"
	"github.com/sourcegraph/makex"
	"sourcegraph.com/sourcegraph/srcgraph/build"
	"sourcegraph.com/sourcegraph/srcgraph/buildstore"
	"sourcegraph.com/sourcegraph/srcgraph/task2"
)

func makefile(args []string) {
	make_cmd(append(args, "-mf"))
}

func make_cmd(args []string) {
	if err := make_(args); err != nil {
		log.Fatal(err)
	}
}

func make_(args []string) error {
	params := mustParseMakeParams(args)
	if err := params.verify(); err != nil {
		return err
	}
	repoConfig := params.RepositoryConfig.GetRepositoryConfig(task2.DefaultContext)

	repoStore, err := buildstore.NewRepositoryStore(params.Repository.RootDir)
	if err != nil {
		return err
	}

	// Get build directory (${REPO}/.sourcegraph-data/...)
	var buildDir string
	if params.Test {
		var err error
		buildDir, err = ioutil.TempDir("", fmt.Sprintf("sourcegraph-data.%s.%s-", strings.Replace(string(repoConfig.URI), "/", "-", -1),
			params.Repository.CommitID))
		if err != nil {
			return err
		}
		if params.TestKeep {
			defer log.Printf("Test build directory: %s", buildDir)
		} else {
			defer os.RemoveAll(buildDir)
		}
	} else {
		buildDir, err = buildstore.BuildDir(repoStore, params.Repository.CommitID)
		if err != nil {
			return err
		}

		// Use a relative base path for the Makefile so that we aren't tied to
		// absolute paths. This makes the Makefile more portable between hosts. (And
		// makex uses vfs, which restricts it to accessing only files under a
		// certain path.)
		buildDir, err = filepath.Rel(params.Repository.RootDir, buildDir)
		if err != nil {
			return err
		}
	}

	// Create Makefile
	mf, err := build.CreateMakefile(buildDir, repoConfig)
	if err != nil {
		return fmt.Errorf("error creating Makefile: %s", err)
	}

	if *Verbose || params.ShowOnly {
		// Show Makefile
		data, err := makex.Marshal(mf)
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		if params.ShowOnly {
			return nil
		}
	}

	// Run Makefile
	err = params.runMakefile(mf)
	if err != nil {
		return err
	}

	if params.Test {
		// Compare expected with actual
		expectedBuildDir, err := buildstore.BuildDir(repoStore, params.Repository.CommitID)
		if err != nil {
			return err
		}

		diffOut, err := exec.Command("diff", "-ur", "--exclude=config.json", expectedBuildDir, buildDir).CombinedOutput()
		log.Print("\n\n\n")
		log.Print("###########################")
		log.Print("##      TEST RESULTS     ##")
		log.Print("###########################")
		if len(diffOut) > 0 {
			diffStr := string(diffOut)
			diffStr = strings.Replace(diffStr, buildDir, "<test-build>", -1)
			log.Printf(diffStr)
			log.Printf(brush.Red("** FAIL **").String())
			return fmt.Errorf("output differed")
		} else if err != nil {
			log.Printf(brush.Red("** ERROR **").String())
			return fmt.Errorf("failed to compute diff: %s", err)
		} else if err == nil {
			log.Printf(brush.Green("** PASS **").String())
			return nil
		}
	}
	return nil
}

type makeParams struct {
	Repository       *repository
	RepositoryConfig *repositoryConfigurator
	Goals            []string

	ShowOnly bool
	Test     bool
	TestKeep bool
	Makex    *makex.Config
}

func mustParseMakeParams(args []string) *makeParams {
	fs := flag.NewFlagSet("make", flag.ExitOnError)
	rc := &repositoryConfigurator{Repository: detectRepository2(*dir)}
	AddRepositoryFlags2(fs, rc.Repository)
	AddRepositoryConfigFlags2(fs, rc)

	showOnly := fs.Bool("mf", false, "print generated Makefile and exit")
	test := fs.Bool("test", false, "test build output against expected output in .sourcegraph-data/")
	testKeep := fs.Bool("test-keep", false, "do NOT delete test build directory after test, used in conjunction with -test")

	conf := &makex.Default
	makex.Flags(fs, conf, "")

	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: `+Name+` make [options] [target...]

Generates and executes a Makefile that processes a repository, creating graph of
definitions, references, and dependencies in a repository's code at a specific
revision.

Run "`+Name+` makefile" to print the generated Makefile and exit.

This command uses makex to execute the Makefile, but the Makefile is also
compatible with GNU make. You can use the "`+Name+` makefile" command to
generate a Makefile to use with GNU make, if you'd like.

The options are:
	   `)
		fs.PrintDefaults()
		fmt.Fprintln(os.Stderr)
		os.Exit(1)
	}
	fs.Parse(args)
	InitializeConfigurator(rc)

	return &makeParams{
		Repository:       rc.Repository,
		RepositoryConfig: rc,
		Goals:            fs.Args(),
		ShowOnly:         *showOnly,
		Test:             *test,
		TestKeep:         *testKeep,
		Makex:            conf,
	}
}

func (p *makeParams) verify() error {
	vcsType := vcs.VCSByName[p.Repository.vcsTypeName]
	if vcsType == nil {
		return fmt.Errorf("%s: unknown VCS type %q", Name, p.Repository.vcsTypeName)
	}
	return nil
}

func (p *makeParams) runMakefile(mf *makex.Makefile) error {
	goals := p.Goals
	if len(goals) == 0 {
		if defaultRule := mf.DefaultRule(); defaultRule != nil {
			goals = []string{defaultRule.Target()}
		} else {
			// No rules in Makefile
			return nil
		}
	}

	err := os.Chdir(p.Repository.RootDir)
	if err != nil {
		return err
	}

	mk := p.Makex.NewMaker(mf, goals...)

	if p.Makex.DryRun {
		err := mk.DryRun(os.Stdout)
		if err != nil {
			return err
		}
		return nil
	}

	err = mk.Run()
	if err != nil {
		return err
	}

	return nil
}
