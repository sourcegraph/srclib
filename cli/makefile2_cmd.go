package cli

import (
	"log"
	"os"

	"sourcegraph.com/sourcegraph/go-flags"

	"sourcegraph.com/sourcegraph/makex"
)

func init() {
	cliInit = append(cliInit, func(cli *flags.Command) {
		_, err := cli.AddCommand("makefile",
			"prints the Makefile that the `make` subcommand executes",
			"The makefile command prints the Makefile that the `make` subcommand will execute.",
			&makefileCmd2,
		)
		if err != nil {
			log.Fatal(err)
		}
	})
}

type MakefileCmd2 struct{}

var makefileCmd2 MakefileCmd2

func (c *MakefileCmd2) Execute(args []string) (err error) {
	mf, err := CreateMakefile2()
	if err != nil {
		return err
	}

	mfData, err := makex.Marshal(mf)
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(mfData)
	return err
}
