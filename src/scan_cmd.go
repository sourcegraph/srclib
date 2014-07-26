package src

import (
	"fmt"
	"log"

	"github.com/sourcegraph/srclib/config"
	"github.com/sourcegraph/srclib/repo"
	"github.com/sourcegraph/srclib/scan"
)

func init() {
	c, err := parser.AddCommand("scan",
		"scan for source units",
		"Scans for source units in the directory tree rooted at the current directory.",
		&scanCmd,
	)
	if err != nil {
		log.Fatal(err)
	}

	setRepoOptDefaults(c)
}

type ScanCmd struct {
	Repo   string `long:"repo" description:"repository URI" value-name:"URI"`
	Subdir string `long:"subdir" description:"subdirectory in repository" value-name:"DIR"`
}

var scanCmd ScanCmd

func (c *ScanCmd) Execute(args []string) error {
	cfg, err := config.ReadRepository(Dir, repo.URI(c.Repo))
	if err != nil {
		return err
	}

	units, err := scan.SourceUnits(Dir, cfg)
	if err != nil {
		return err
	}

	for _, u := range units {
		fmt.Println(u.ID())
	}

	return nil
}
