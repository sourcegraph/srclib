package cli

import (
	"log"

	"sourcegraph.com/sourcegraph/srclib/gendata"
)

func init() {
	c, err := CLI.AddCommand("gen-data",
		"generates fake data",
		`generates fake data for testing and benchmarking purposes. Run this command inside an empty or expendable directory.`,
		&gendata.GenDataCmd{},
	)
	if err != nil {
		log.Fatal(err)
	}
	c.Aliases = []string{"c"}

	_, err = c.AddCommand("simple",
		"generates a simple repository",
		"generates a simple repository with the specified source unit and file structure, with the given number of defs and refs to those defs in each file",
		&gendata.SimpleRepoCmd{},
	)
	if err != nil {
		log.Fatal(err)
	}
}
