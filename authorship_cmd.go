package srcgraph

import (
	"flag"
	"fmt"
	"log"
	"os"

	"sourcegraph.com/sourcegraph/srcgraph/authorship"
	"sourcegraph.com/sourcegraph/srcgraph/grapher2"
	"sourcegraph.com/sourcegraph/srcgraph/vcsutil"
)

func authorship_(args []string) {
	fs := flag.NewFlagSet("authorship", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: `+Name+` authorship [options] blame.json graph.json

Determines who authored a source unit's symbols and refs in a graph output file
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

	context, err := NewJobContext(*Dir)
	if err != nil {
		log.Fatal(err)
	}

	var b *vcsutil.BlameOutput
	readJSONFile(blameFile, &b)

	var g *grapher2.Output
	readJSONFile(graphFile, &g)

	out, err := authorship.ComputeSourceUnit(g, b, context.Repo)
	if err != nil {
		log.Fatal(err)
	}

	PrintJSON(out, "")
}
