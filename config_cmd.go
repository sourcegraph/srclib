package srcgraph

import (
	"flag"
	"fmt"
	"log"
	"os"

	"sourcegraph.com/sourcegraph/srcgraph/task2"
)

func config_(args []string) {
	fs := flag.NewFlagSet("config", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: `+Name+` config [options]

Validates and prints a repository's configuration.

The options are:
`)
		fs.PrintDefaults()
		os.Exit(1)
	}
	fs.Parse(args)

	context, err := NewJobContext(*Dir, task2.DefaultContext)
	if err != nil {
		log.Fatal(err)
	}

	PrintJSON(context.Repo, "")
}
