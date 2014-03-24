package build

import (
	"strconv"

	"sourcegraph.com/sourcegraph/srcgraph/config"
	_ "sourcegraph.com/sourcegraph/srcgraph/toolchain/all_toolchains"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
	"sourcegraph.com/sourcegraph/srcgraph/util2/makefile"
)

func init() {
	RegisterRuleMaker("graph", makeGraphRules)
}

func makeGraphRules(c *config.Repository, existing []makefile.Rule) ([]makefile.Rule, error) {
	var rules []makefile.Rule
	for _, u := range c.SourceUnits {
		rules = append(rules, &GraphSourceUnitRule{u})
	}
	return rules, nil
}

type GraphSourceUnitRule struct {
	Unit unit.SourceUnit
}

func (r *GraphSourceUnitRule) Target() makefile.Target {
	return &SourceUnitOutputFile{r.Unit, "graph"}
}

func (r *GraphSourceUnitRule) Prereqs() []string { return r.Unit.Paths() }

func (r *GraphSourceUnitRule) Recipes() []makefile.Recipe {
	return []makefile.Recipe{
		makefile.CommandRecipe{"srcgraph", "-v", "graph", "-json", strconv.Quote(string(unit.MakeID(r.Unit))), "1> $@"},
	}
}
