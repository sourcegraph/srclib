package srcgraph

import (
	"flag"
	"fmt"
	"log"
	"os"

	"sourcegraph.com/sourcegraph/srcgraph/task2"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
	"sourcegraph.com/sourcegraph/srcgraph/vcsutil"
)

func blame(args []string) {
	fs := flag.NewFlagSet("blame", flag.ExitOnError)
	r := AddRepositoryFlags(fs)
	rc := AddRepositoryConfigFlags(fs, r)
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

	c := rc.GetRepositoryConfig(task2.DefaultContext)

	for _, u := range c.SourceUnits {
		if !SourceUnitMatchesArgs(sourceUnitSpecs, u) {
			continue
		}

		paths, err := unit.ExpandPaths(r.RootDir, u.Paths())
		if err != nil {
			log.Fatal(err)
		}

		out, err := vcsutil.BlameFiles(r.RootDir, paths, r.CommitID, c, task2.DefaultContext)
		if err != nil {
			log.Fatal(err)
		}

		PrintJSON(out, "")
	}
}
