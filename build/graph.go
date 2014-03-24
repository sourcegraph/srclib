package build

import (
	"path/filepath"

	"strconv"

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

type GraphSourceUnitRule struct {
	SourceUnitSpec
}

func (r *GraphSourceUnitRule) Target() makefile.Target {
	return &SourceUnitOutputFile{r.SourceUnitSpec, "graph"}
}

func (r *GraphSourceUnitRule) Prereqs() []string { return r.Unit.Paths() }

func (r *GraphSourceUnitRule) Recipes() []makefile.Recipe {
	return []makefile.Recipe{
		makefile.CommandRecipe{"mkdir", "-p", strconv.Quote(filepath.Dir(r.Target().Name()))},
		makefile.CommandRecipe{"srcgraph", "-v", "graph", "-json", strconv.Quote(string(unit.MakeID(r.Unit))), "1> $@"},
	}
}
