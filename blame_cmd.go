package srcgraph

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
	"sourcegraph.com/sourcegraph/srcgraph/vcsutil"
)

func blame(args []string) {
	fs := flag.NewFlagSet("blame", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: `+Name+` blame [options] [unit...]

Blames the files in a source unit. If no source units are specified, all source
units are blamed.

The options are:
`)
		fs.PrintDefaults()
		os.Exit(1)
	}
	fs.Parse(args)
	sourceUnitSpecs := fs.Args()

	context, err := NewJobContext(*Dir)
	if err != nil {
		log.Fatal(err)
	}

	for _, u := range context.Repo.SourceUnits {
		if !SourceUnitMatchesArgs(sourceUnitSpecs, u) {
			continue
		}

		paths, err := unit.ExpandPaths(context.RepoRootDir, u.Paths())
		if err != nil {
			log.Fatal(err)
		}

		var out *vcsutil.BlameOutput
		if paths == nil {
			out, err = vcsutil.BlameRepository(context.RepoRootDir, context.CommitID, context.Repo)
		} else {
			out, err = vcsutil.BlameFiles(context.RepoRootDir, paths, context.CommitID, context.Repo)
		}
		if err != nil {
			log.Fatal(err)
		}
		PrintJSON(out, "")
	}
}
