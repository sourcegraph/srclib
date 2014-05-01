package srcgraph

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/sourcegraph/makex"
	"sourcegraph.com/sourcegraph/srcgraph/build"
	"sourcegraph.com/sourcegraph/srcgraph/buildstore"
	"sourcegraph.com/sourcegraph/srcgraph/task2"
)

func makefile(args []string) {
	make_(append(args, "-mf"))
}

func make_(args []string) {
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

	context, err := NewJobContext(*Dir, task2.DefaultContext)
	if err != nil {
		log.Fatal(err)
	}

	err = make__(goals, context, conf, *showOnly, *Verbose)
	if err != nil {
		log.Fatal(err)
	}
}

func make__(goals []string, context *JobContext, conf *makex.Config, showOnly bool, verbose bool) error {
	if err := WriteRepositoryConfig(context.RepoRootDir, context.CommitID, context.Repo, false); err != nil {
		return fmt.Errorf("unable to write repository config file due to error %s", err)
	}

	repoStore, err := buildstore.NewRepositoryStore(context.RepoRootDir)
	if err != nil {
		return err
	}
	buildDir, err := buildstore.BuildDir(repoStore, context.CommitID)
	if err != nil {
		return err
	}
	// Use a relative base path for the Makefile so that we aren't tied to
	// absolute paths. This makes the Makefile more portable between hosts. (And
	// makex uses vfs, which restricts it to accessing only files under a
	// certain path.)
	buildDir, err = filepath.Rel(context.RepoRootDir, buildDir)
	if err != nil {
		return err
	}

	mf, err := build.CreateMakefile(buildDir, context.Repo)
	if err != nil {
		return err
	}

	if verbose || showOnly {
		data, err := makex.Marshal(mf)
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		if showOnly {
			return nil
		}
	}

	// Run Makefile
	err = runMakefile(mf, conf, context.RepoRootDir, goals)
	if err != nil {
		return err
	}
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
