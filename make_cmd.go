package srcgraph

import (
	"flag"
	"fmt"
	"log"
	"os"
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
	fs := flag.NewFlagSet("make", flag.ExitOnError)
	showOnly := fs.Bool("mf", false, "print generated makefile and exit")
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
	repoDir := *dir

	context, err := NewJobContext(repoDir, task2.DefaultContext)
	if err != nil {
		return err
	}

	repoStore, err := buildstore.NewRepositoryStore(context.RepoRootDir)
	if err != nil {
		return err
	}
	buildDir, err := buildstore.BuildDir(repoStore, context.CommitID)
	if err != nil {
		return err
	}

	mf, err := build.CreateMakefile(buildDir, context.Repo)
	if err != nil {
		return err
	}

	if *Verbose || *showOnly {
		data, err := makex.Marshal(mf)
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		if *showOnly {
			return nil
		}
	}

	// Run Makefile
	err = runMakefile(mf, conf, repoDir, goals)
	if err != nil {
		return err
	}

	// TODO: support test case

	// if params.Test {
	// 	// Compare expected with actual
	// 	expectedBuildDir, err := buildstore.BuildDir(repoStore, params.Repository.CommitID)
	// 	if err != nil {
	// 		return err
	// 	}

	// 	diffOut, err := exec.Command("diff", "-ur", "--exclude=config.json", expectedBuildDir, buildDir).CombinedOutput()
	// 	log.Print("\n\n\n")
	// 	log.Print("###########################")
	// 	log.Print("##      TEST RESULTS     ##")
	// 	log.Print("###########################")
	// 	if len(diffOut) > 0 {
	// 		diffStr := string(diffOut)
	// 		diffStr = strings.Replace(diffStr, buildDir, "<test-build>", -1)
	// 		log.Printf(diffStr)
	// 		log.Printf(brush.Red("** FAIL **").String())
	// 		return fmt.Errorf("output differed")
	// 	} else if err != nil {
	// 		log.Printf(brush.Red("** ERROR **").String())
	// 		return fmt.Errorf("failed to compute diff: %s", err)
	// 	} else if err == nil {
	// 		log.Printf(brush.Green("** PASS **").String())
	// 		return nil
	// 	}
	// }

	return nil
}

func runMakefile(mf *makex.Makefile, conf *makex.Config, repoDir string, goals []string) error {
	if len(goals) == 0 {
		if defaultRule := mf.DefaultRule(); defaultRule != nil {
			goals = []string{defaultRule.Target()}
		} else {
			// No rules in Makefile
			return nil
		}
	}

	err := os.Chdir(repoDir) // TODO: kinda ugly
	if err != nil {
		return err
	}
	mk := conf.NewMaker(mf, goals...)
	if conf.DryRun {
		return mk.DryRun(os.Stdout)
	}
	return mk.Run()
}
