package srcgraph

import (
	"flag"
	"fmt"
	"log"
	"os"

	"sourcegraph.com/sourcegraph/repo"
	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/scan"
	"sourcegraph.com/sourcegraph/srcgraph/task2"
)

func config_(args []string) {
	fs := flag.NewFlagSet("config", flag.ExitOnError)
	r := AddRepositoryFlags(fs)
	final := fs.Bool("final", true, "add scanned source units and finalize config before printing")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: `+Name+` config [options]

Validates and prints a repository's configuration.

The options are:
`)
		fs.PrintDefaults()
		os.Exit(1)
	}
	fs.Parse(args)

	repoURI := repo.MakeURI(r.CloneURL)

	x := task2.DefaultContext

	var c *config.Repository
	var err error
	if *final {
		c, err = scan.ReadDirConfigAndScan(r.RootDir, repoURI, x)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		c, err = config.ReadDir(r.RootDir, repoURI)
		if err != nil {
			log.Fatal(err)
		}
	}

	PrintJSON(c, "")
}
