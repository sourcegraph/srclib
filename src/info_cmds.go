package src

import (
	"log"

	"github.com/sourcegraph/srclib/build"
	"github.com/sourcegraph/srclib/buildstore"
	"github.com/sourcegraph/srclib/toolchain"
)

func init() {
	_, err := CLI.AddCommand("info",
		"show info about enabled capabilities",
		"Shows information about enabled capabilities in this tool as well as system information.",
		&infoCmd,
	)
	if err != nil {
		log.Fatal(err)
	}
}

type InfoCmd struct{}

var infoCmd InfoCmd

func (c *InfoCmd) Execute(args []string) error {
	log.Printf("srclib version %s\n", Version)
	log.Println("https://sourcegraph.com/sourcegraph/srclib")
	log.Println()

	log.Printf("SRCLIBPATH=%q", toolchain.SrclibPath)

	log.Println()
	log.Printf("Build data types (%d)", len(buildstore.DataTypes))
	for name, _ := range buildstore.DataTypes {
		log.Printf(" - %s", name)
	}
	log.Println()

	log.Printf("Build rule makers (%d)", len(build.RuleMakers))
	for name, _ := range build.RuleMakers {
		log.Printf(" - %s", name)
	}

	return nil
}
