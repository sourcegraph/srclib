package build

import (
	"strconv"

	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
	"sourcegraph.com/sourcegraph/srcgraph/util2/makefile"
)

func init() {
	RegisterRuleMaker("dep", makeDepRules)
}

func makeDepRules(c *config.Repository, existing []makefile.Rule) ([]makefile.Rule, error) {
	if len(c.SourceUnits) == 0 {
		return nil, nil
	}

	resolveRule := &ResolveDepsRule{}

	rules := []makefile.Rule{resolveRule}
	for _, u := range c.SourceUnits {
		rule := &ListSourceUnitDepsRule{u}
		rules = append(rules, rule)
		resolveRule.RawDepLists = append(resolveRule.RawDepLists, rule.Target())
	}

	return rules, nil
}

type ResolveDepsRule struct {
	RawDepLists []makefile.Target
}

func (r *ResolveDepsRule) Target() makefile.Target {
	return &RepositoryCommitOutputFile{"resolved_deps"}
}

func (r *ResolveDepsRule) Prereqs() []string {
	var files []string
	for _, rawDepListFile := range r.RawDepLists {
		files = append(files, rawDepListFile.Name())
	}
	return files
}

func (r *ResolveDepsRule) Recipes() []makefile.Recipe {
	return []makefile.Recipe{
		makefile.CommandRecipe{"srcgraph", "-v", "resolve-deps", "-json", "$^", "1> $@"},
	}
}

type ListSourceUnitDepsRule struct {
	Unit unit.SourceUnit
}

func (r *ListSourceUnitDepsRule) Target() makefile.Target {
	return &SourceUnitOutputFile{r.Unit, "raw_deps"}
}

func (r *ListSourceUnitDepsRule) Prereqs() []string { return r.Unit.Paths() }

func (r *ListSourceUnitDepsRule) Recipes() []makefile.Recipe {
	return []makefile.Recipe{
		makefile.CommandRecipe{"srcgraph", "-v", "list-deps", "-json", strconv.Quote(string(unit.MakeID(r.Unit))), "1> $@"},
	}
}
