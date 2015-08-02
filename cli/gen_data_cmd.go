package cli

import (
	"log"

	"sourcegraph.com/sourcegraph/srclib/dep"
	"sourcegraph.com/sourcegraph/srclib/gendata"
	"sourcegraph.com/sourcegraph/srclib/graph"
	"sourcegraph.com/sourcegraph/srclib/unit"
)

func init() {
	c, err := CLI.AddCommand("gen-data",
		"generates fake data",
		`generates fake data for testing and benchmarking purposes. Run this command inside an empty or expendable directory.`,
		&genDataCmd,
	)
	if err != nil {
		log.Fatal(err)
	}
	c.Aliases = []string{"c"}
}

type GenDataCmd struct {
	Repo     string `short:"r" long:"repo" description:"repo to build" required:"yes"`
	CommitID string `short:"c" long:"commit" description:"commit ID to build"`
	NFiles   []int  `short:"f" long:"files" description:"number of files at each level" required:"yes"`
	NUnits   []int  `short:"u" long:"units" description:"number of units to generate; uses same input structure as --files" required:"yes"`
	NDefs    int    `long:"ndefs" description:"number of defs to generate per file" required:"yes"`
	NRefs    int    `long:"nrefs" description:"number of refs to generate per file" required:"yes"`

	GenSource bool `long:"gen-source" description:"whether to emit source files for the generated data"`
}

var genDataCmd GenDataCmd

type unitInfo struct {
	Unit  *unit.SourceUnit
	Graph *graph.Output
	Deps  []*dep.Resolution
}

func (c *GenDataCmd) Execute(args []string) error {
	cfg := gendata.GenDataCmd{
		Repo:      c.Repo,
		CommitID:  c.CommitID,
		NFiles:    c.NFiles,
		NUnits:    c.NUnits,
		NDefs:     c.NDefs,
		NRefs:     c.NRefs,
		GenSource: c.GenSource,
	}
	return cfg.Generate()
}
