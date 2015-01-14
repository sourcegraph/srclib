package src

import (
	"encoding/json"
	"log"
	"os"

	"sourcegraph.com/sourcegraph/srclib"
	"sourcegraph.com/sourcegraph/srclib/buildstore"
	"sourcegraph.com/sourcegraph/srclib/config"
	"sourcegraph.com/sourcegraph/srclib/graph"
	"sourcegraph.com/sourcegraph/srclib/graphstore"
)

func init() {
	c, err := CLI.AddCommand("store",
		"graph store commands",
		"",
		&storeCmd,
	)
	if err != nil {
		log.Fatal(err)
	}

	_, err = c.AddCommand("import",
		"import data",
		`Import data into the graph store. Only imports ref data for now.`,
		&storeImportCmd,
	)
	if err != nil {
		log.Fatal(err)
	}

	_, err = c.AddCommand("listrefs",
		"list all the refs for a defkey",
		"Return all the references for the specified def key.",
		&listRefsCmd,
	)
	if err != nil {
		log.Fatal(err)
	}
}

var graphStore *graphstore.Store

func init() {
	var err error
	graphStore, err = graphstore.NewLocal(srclib.Path)
	if err != nil {
		log.Fatal(err)
	}
}

type StoreCmd struct{}

var storeCmd StoreCmd

func (c *StoreCmd) Execute(args []string) error { return nil }

type StoreImportCmd struct {
	config.Options
}

var storeImportCmd StoreImportCmd

func (c *StoreImportCmd) Execute(args []string) error {
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
	gs, err := graphstore.NewLocal(srclib.Path)
	if err != nil {
		return err
	}
	if err := gs.StoreRefs(output.Refs); err != nil {
		return err
	}
	return nil
}

type ListRefsCmd struct {
	Repo     string `long:"repo" required:"true"`
	UnitType string `long:"unittype" required:"true"`
	Unit     string `long:"unit" required:"true"`
	Path     string `long:"path" required:"true"`

	// If RefsRepo is not empty, only fetch refs from this
	// repository URI.
	RefsRepo string `long:"refrepo"`
}

var listRefsCmd ListRefsCmd

func (c *ListRefsCmd) Execute(args []string) error {
	dk := graph.DefKey{
		Repo:     c.Repo,
		UnitType: c.UnitType,
		Unit:     c.Unit,
		Path:     graph.DefPath(c.Path),
	}
	refs, err := graphStore.ListRefs(dk, &graphstore.ListRefsOptions{Repo: c.RefsRepo})
	if err != nil {
		return err
	}
	if err := json.NewEncoder(os.Stdout).Encode(refs); err != nil {
		return err
	}
	return nil
}
