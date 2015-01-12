package src

import (
	"log"

	"sourcegraph.com/sourcegraph/srclib"
	"sourcegraph.com/sourcegraph/srclib/buildstore"
	"sourcegraph.com/sourcegraph/srclib/config"
	"sourcegraph.com/sourcegraph/srclib/graphstore"
)

func init() {
	_, err := CLI.AddCommand("import",
		"import data",
		`Import data into the graph store. Only imports ref data for now.`,
		&importCmd,
	)
	if err != nil {
		log.Fatal(err)
	}
}

type ImportCmd struct {
	config.Options
}

var importCmd ImportCmd

func (c *ImportCmd) Execute(args []string) error {
	// TODO(samer): honor options.
	localRepo, err := OpenRepo(".")
	if err != nil {
		return err
	}
	buildStore, err := buildstore.LocalRepo(localRepo.RootDir)
	if err != nil {
		return err
	}
	output, err := config.ReadCachedGraph(buildStore.Commit(localRepo.CommitID))
	if err != nil {
		return err
	}
	if len(output.Refs) == 0 {
		log.Println("No refs found.")
		// This is not an error, but there is no more work to
		// be done.
		return nil
	}
	gs, err := graphstore.New(srclib.Path)
	if err != nil {
		return err
	}
	if err := gs.StoreRefs(output.Refs); err != nil {
		return err
	}
	return nil
}
