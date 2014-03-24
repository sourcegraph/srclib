package srcgraph

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/sourcegraph/go-vcs"
	"sourcegraph.com/sourcegraph/srcgraph/build"
	"sourcegraph.com/sourcegraph/srcgraph/task2"
	"sourcegraph.com/sourcegraph/srcgraph/util2/makefile"
)

func build_(args []string) {
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	repo := AddRepositoryFlags(fs)
	dryRun := fs.Bool("n", false, "dry run (scans the repository and just prints out what analysis tasks would be performed)")
	outputFile := fs.String("o", "", "write output to file")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: `+Name+` build [options]

Builds a graph of definitions, references, and dependencies in a repository's
code at a specific revision.

The options are:
`)
		fs.PrintDefaults()
		os.Exit(1)
	}
	fs.Parse(args)

	if *outputFile == "" {
		*outputFile = repo.outputFile()
	}

	build.WorkDir = *tmpDir
	mkTmpDir()

	if fs.NArg() != 0 {
		fs.Usage()
	}

	vcsType := vcs.VCSByName[repo.vcsTypeName]
	if vcsType == nil {
		log.Fatalf("%s: unknown VCS type %q", Name, repo.vcsTypeName)
	}

	x := task2.NewRecordedContext()

	rules, err := build.CreateMakefile(repo.rootDir, repo.CloneURL, repo.commitID, x)
	if err != nil {
		log.Fatalf("error creating Makefile: %s", err)
	}

	if *verbose || *dryRun {
		mf, err := makefile.Makefile(rules)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("# Makefile\n%s", mf)
	}
	if *dryRun {
		return
	}

	err = makefile.MakeRules(repo.rootDir, rules)
	if err != nil {
		log.Fatalf("make failed: %s", err)
	}

	if *verbose {
		if len(rules) > 0 {
			log.Printf("%d output files:", len(rules))
			for _, r := range rules {
				log.Printf(" - %s", r.Target().Name())
			}
		}
	}
}
