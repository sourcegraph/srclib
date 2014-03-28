package build

import (
	"fmt"
	"path/filepath"

	"sourcegraph.com/sourcegraph/srcgraph/buildstore"
	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/dep2"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
	"sourcegraph.com/sourcegraph/srcgraph/util2/makefile"
)

func init() {
	RegisterRuleMaker("dep", makeDepRules)
	buildstore.RegisterDataType("raw_deps.v0", []*dep2.RawDependency{})
	buildstore.RegisterDataType("resolved_deps.v0", []*dep2.ResolvedDep{})
}

func makeDepRules(c *config.Repository, dataDir string, existing []makefile.Rule) ([]makefile.Rule, error) {
	if len(c.SourceUnits) == 0 {
		return nil, nil
	}

	var rules []makefile.Rule
	for _, u := range c.SourceUnits {
		rawDepRule := &ListSourceUnitDepsRule{dataDir, u}
		rules = append(rules, rawDepRule)
		rules = append(rules, &ResolveDepsRule{dataDir, u, rawDepRule.Target()})
	}

	return rules, nil
}

type ResolveDepsRule struct {
	dataDir       string
	unit          unit.SourceUnit
	RawDepsOutput string
}

func (r *ResolveDepsRule) Target() string {
	return filepath.Join(r.dataDir, SourceUnitDataFilename([]*dep2.ResolvedDep{}, r.unit))
}

func (r *ResolveDepsRule) Prereqs() []string { return []string{r.RawDepsOutput} }

func (r *ResolveDepsRule) Recipes() []string {
	return []string{"srcgraph -v resolve-deps -json $^ 1> $@"}
}

type ListSourceUnitDepsRule struct {
	dataDir string
	unit    unit.SourceUnit
}

func (r *ListSourceUnitDepsRule) Target() string {
	return filepath.Join(r.dataDir, SourceUnitDataFilename([]*dep2.RawDependency{}, r.unit))
}

func (r *ListSourceUnitDepsRule) Prereqs() []string {
	return r.unit.Paths()
}

func (r *ListSourceUnitDepsRule) Recipes() []string {
	return []string{
		"mkdir -p `dirname \"$@\"`",
		fmt.Sprintf("srcgraph -v list-deps -json %q 1> $@", unit.MakeID(r.unit)),
	}
}
