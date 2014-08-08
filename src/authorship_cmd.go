//+build off

package src

import (
	"flag"
	"fmt"
	"log"
	"os"

	"sourcegraph.com/sourcegraph/srclib/authorship"
	"sourcegraph.com/sourcegraph/srclib/grapher"
	"sourcegraph.com/sourcegraph/srclib/vcsutil"
)

func authorship_(args []string) {
	fs := flag.NewFlagSet("authorship", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: `+Name+` authorship [options] blame.json graph.json

Determines who authored a source unit's defs and refs in a graph output file
by merging it with VCS blame info.

The options are:
`)
		fs.PrintDefaults()
		os.Exit(1)
	}
	fs.Parse(args)
	if fs.NArg() != 2 {
		fs.Usage()
	}
	blameFile, graphFile := fs.Arg(0), fs.Arg(1)

	repoConf, err := OpenAndConfigureRepo(Dir)
	if err != nil {
		log.Fatal(err)
	}

	var b *vcsutil.BlameOutput
	readJSONFile(blameFile, &b)

	var g *grapher.Output
	readJSONFile(graphFile, &g)

	out, err := authorship.ComputeSourceUnit(g, b, repoConf.Config)
	if err != nil {
		log.Fatal(err)
	}

	PrintJSON(out, "")
}
