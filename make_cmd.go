package srcgraph

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/aybabtme/color/brush"
	"github.com/sourcegraph/makex"
	"sourcegraph.com/sourcegraph/srcgraph/build"
	"sourcegraph.com/sourcegraph/srcgraph/buildstore"
)

var testBuildDataDirName = buildstore.BuildDataDirName + "-exp"

func makefile(args []string) {
	make_(append(args, "-mf"))
}

func make_(args []string) {
	fs := flag.NewFlagSet("make", flag.ExitOnError)
	showOnly := fs.Bool("mf", false, "print generated makefile and exit")
	test := fs.Bool("test", false, "diff against expected test data")
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
	goals := fs.Args()

	context, err := NewJobContext(*Dir)
	if err != nil {
		log.Fatal(err)
	}

	if *test {
		context.CommitID = "test-commit"
	}

	mk, mf, err := NewMaker(goals, context, conf)
	if err != nil {
		log.Fatal(err)
	}

	if *Verbose || *showOnly {
		data, err := makex.Marshal(mf)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(data))
		if *showOnly {
			return
		}
	}

	if conf.DryRun {
		err := mk.DryRun(os.Stdout)
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	err = mk.Run()
	if err != nil {
		log.Fatal(err)
	}

	if *test {
		success := compareResults(testBuildDataDirName, buildstore.BuildDataDirName)
		if !success {
			os.Exit(1)
		}
	}
}

func NewMaker(goals []string, context *JobContext, conf *makex.Config) (*makex.Maker, *makex.Makefile, error) {
	if err := WriteRepositoryConfig(context.RepoRootDir, context.CommitID, context.Repo, false); err != nil {
		return nil, nil, fmt.Errorf("unable to write repository config file due to error %s", err)
	}

	repoStore, err := buildstore.NewRepositoryStore(context.RepoRootDir)
	if err != nil {
		return nil, nil, err
	}
	buildDir, err := buildstore.BuildDir(repoStore, context.CommitID)
	if err != nil {
		return nil, nil, err
	}
	// Use a relative base path for the Makefile so that we aren't tied to
	// absolute paths. This makes the Makefile more portable between hosts. (And
	// makex uses vfs, which restricts it to accessing only files under a
	// certain path.)
	buildDir, err = filepath.Rel(context.RepoRootDir, buildDir)
	if err != nil {
		return nil, nil, err
	}

	mf, err := build.CreateMakefile(buildDir, context.Repo)
	if err != nil {
		return nil, nil, err
	}

	if len(goals) == 0 {
		if defaultRule := mf.DefaultRule(); defaultRule != nil {
			goals = []string{defaultRule.Target()}
		}
	}

	// Change to the directory that make prereqs are relative to, so that makex
	// can properly compute the DAG.
	if err := os.Chdir(context.RepoRootDir); err != nil {
		return nil, nil, err
	}

	return conf.NewMaker(mf, goals...), mf, nil
}

func compareResults(expDir, actDir string) bool {
	diffOut, err := exec.Command("diff", "-ur", expDir, actDir).CombinedOutput()
	if err != nil {
		log.Printf(brush.Red("ERROR").String())
		log.Printf("diff failed (%s), diff output: %s", err, string(diffOut))
		return false
	} else if len(diffOut) > 0 {
		diffStr := string(diffOut)
		log.Printf(brush.Red("FAIL").String())
		log.Printf(diffStr)
		log.Printf("output differed")
		return false
	} else {
		log.Printf(brush.Green("PASS").String())
		return true
	}
}

// func runMakefile(mf *makex.Makefile, conf *makex.Config, repoDir string, goals []string, w io.Writer) error {

// 	mk.RuleOutput = func(rule makex.Rule) (out io.Writer, err io.Writer) {
// 		return nil, nil
// 	}

// 	return mk.Run()
// }
