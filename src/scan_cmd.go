package src

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/sourcegraph/srclib/repo"
	"github.com/sourcegraph/srclib/scan"
)

func init() {
	parser.AddCommand("scan",
		"scan for source units",
		"Long description",
		&scanCmd,
	)
}

type ScanCmd struct {
	Repo string `long:"repo" description:"repository URI" value-name:"URI"`
}

var scanCmd ScanCmd

func (c *ScanCmd) Execute(args []string) error {
	return nil
}

func scanCmd2(args []string) {
	repo_, err := OpenRepo(Dir)
	if err != nil {
		log.Fatal(err)
	}

	fs := flag.NewFlagSet("scan", flag.ExitOnError)
	repoURI := fs.String("repo", string(repo_.URI()), "repository URI")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: `+Name+` scan [options]

Scans for source units in the directory tree rooted at the current directory.

The options are:
`)
		fs.PrintDefaults()
		os.Exit(1)
	}
	fs.Parse(args)

	c, err := scan.ReadRepositoryAndScan(Dir, repo.URI(*repoURI))
	if err != nil {
		log.Fatal(err)
	}

	for _, u := range c.SourceUnits {
		fmt.Println(u.ID())
	}
}
