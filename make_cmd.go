package srcgraph

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/sourcegraph/go-vcs"
	"github.com/sourcegraph/makex"
	"sourcegraph.com/sourcegraph/srcgraph/build"
	"sourcegraph.com/sourcegraph/srcgraph/task2"
)

func makefile(args []string) {
	make_(append(args, "-mf"))
}

func make_(args []string) {
	fs := flag.NewFlagSet("make", flag.ExitOnError)
	r := AddRepositoryFlags(fs)
	rc := AddRepositoryConfigFlags(fs, r)
	showMakefileAndExit := fs.Bool("mf", false, "print generated Makefile and exit")
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

	vcsType := vcs.VCSByName[r.vcsTypeName]
	if vcsType == nil {
		log.Fatalf("%s: unknown VCS type %q", Name, r.vcsTypeName)
	}

	c := rc.GetRepositoryConfig(task2.DefaultContext)

	mf, err := build.CreateMakefile(r.RootDir, r.CommitID, c, conf, task2.DefaultContext)
	if err != nil {
		log.Fatalf("error creating Makefile: %s", err)
	}

	if *Verbose || *showMakefileAndExit {
		data, err := makex.Marshal(mf)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(data))
		if *showMakefileAndExit {
			return
		}
	}

	goals := fs.Args()
	if len(goals) == 0 {
		if defaultRule := mf.DefaultRule(); defaultRule != nil {
			goals = []string{defaultRule.Target()}
		} else {
			log.Println("No rules in Makefile.")
			return
		}
	}

	mk := conf.NewMaker(mf, goals...)

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
}
