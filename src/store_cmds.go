package src

import (
	"encoding/json"
	"log"
	"os"

	"sourcegraph.com/sourcegraph/srclib"
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
	graphStore, err = graphstore.New(srclib.Path)
	if err != nil {
		log.Fatal(err)
	}
}

type StoreCmd struct{}

var storeCmd StoreCmd

func (c *StoreCmd) Execute(args []string) error { return nil }

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
