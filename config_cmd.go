package srcgraph

import (
	"flag"
	"fmt"
	"os"
	"sourcegraph.com/sourcegraph/srcgraph/task2"
)

func config_(args []string) {
	fs := flag.NewFlagSet("config", flag.ExitOnError)
	r := AddRepositoryFlags(fs)
	rc := AddRepositoryConfigFlags(fs, r)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: `+Name+` config [options]

Validates and prints a repository's configuration.

The options are:
`)
		fs.PrintDefaults()
		os.Exit(1)
	}
	fs.Parse(args)

	c := rc.GetRepositoryConfig(task2.DefaultContext)

	PrintJSON(c, "")
}
