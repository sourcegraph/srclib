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

func make_(args []string) {
	fs := flag.NewFlagSet("make", flag.ExitOnError)
	repo := AddRepositoryFlags(fs)
	showMakefileAndExit := fs.Bool("mf", false, "print generated Makefile and exit")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: `+Name+` make [options] [-- makeoptions] [target...]

Generates a Makefile that processes a repository, creating graph of definitions,
references, and dependencies in a repository's code at a specific revision.

Targets and extra options (after "--") are passed directly to the "make"
program, which executes the generated Makefile. If no targets are specified,
"all" is built.

The options are:
`)
		fs.PrintDefaults()
		fmt.Fprintln(os.Stderr, `
The most useful makeoptions are:

    -n, --dry-run       don't actually run any commands (just print them)
    -k, --keep-going    keep going when some targets can't be made
    -j N, --jobs N      allow N parallel jobs

See the man page for "make" for all makeoptions.
`)
		os.Exit(1)
	}
	fs.Parse(args)

	build.WorkDir = *tmpDir
	mkTmpDir()

	vcsType := vcs.VCSByName[repo.vcsTypeName]
	if vcsType == nil {
		log.Fatalf("%s: unknown VCS type %q", Name, repo.vcsTypeName)
	}

	x := task2.NewRecordedContext()

	rules, vars, err := build.CreateMakefile(repo.rootDir, repo.CloneURL, repo.commitID, x)
	if err != nil {
		log.Fatalf("error creating Makefile: %s", err)
	}

	if *verbose || *showMakefileAndExit {
		mf, err := makefile.Makefile(rules, vars)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("# Makefile\n%s", mf)
		if *showMakefileAndExit {
			return
		}
	}

	err = makefile.MakeRules(repo.rootDir, rules, vars, fs.Args())
	if err != nil {
		log.Fatalf("make failed: %s", err)
	}

	if *verbose {
		if len(rules) > 0 {
			log.Printf("%d output files:", len(rules))
			for _, r := range rules {
				log.Printf(" - %s", build.SubstituteVars(r.Target().Name(), vars))
			}
		}
	}
}
