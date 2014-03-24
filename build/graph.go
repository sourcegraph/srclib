package build

import (
	"path/filepath"

	"strconv"

	"sourcegraph.com/sourcegraph/repo"
	"sourcegraph.com/sourcegraph/srcgraph/config"
	_ "sourcegraph.com/sourcegraph/srcgraph/toolchain/all_toolchains"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
	"sourcegraph.com/sourcegraph/srcgraph/util2/makefile"
)

func init() {
	RegisterRuleMaker("graph", makeGraphRules)
}

func makeGraphRules(c *config.Repository, commitID string) ([]makefile.Rule, error) {
	var rules []makefile.Rule
	for _, u := range c.SourceUnits {
		us := SourceUnitSpec{
			RepositoryURI: c.URI,
			CommitID:      commitID,
			Unit:          u,
		}
		rules = append(rules, &GraphSourceUnitRule{us})
	}
	return rules, nil
}

type SourceUnitSpec struct {
	RepositoryURI repo.URI
	CommitID      string
	Unit          unit.SourceUnit
}

type GraphSourceUnitRule struct {
	SourceUnitSpec
}

func (r *GraphSourceUnitRule) Target() makefile.Target {
	return &GraphOutputFile{r.SourceUnitSpec}
}

func (r *GraphSourceUnitRule) Prereqs() []string {
	return r.Unit.Paths()
}

func (r *GraphSourceUnitRule) Recipes() []makefile.Recipe {
	return []makefile.Recipe{
		makefile.CommandRecipe{"mkdir", "-p", strconv.Quote(filepath.Dir(r.Target().Name()))},
		makefile.CommandRecipe{"srcgraph", "-v", "graph", "-json", strconv.Quote(r.Unit.ID()), "1> $@"},
	}
}

type GraphOutputFile struct {
	SourceUnitSpec
}

func (f *GraphOutputFile) Name() string {
	return filepath.Join(WorkDir, string(f.RepositoryURI), f.CommitID, f.RelName())
}

func (f *GraphOutputFile) RelName() string {
	return filepath.Join(f.Unit.RootDir(), f.Unit.ID()+"_graph.json")
}
