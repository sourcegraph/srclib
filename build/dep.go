package build

import (
	"fmt"
	"reflect"

	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/dep2"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
	"sourcegraph.com/sourcegraph/srcgraph/util2/makefile"
)

func init() {
	RegisterRuleMaker("dep", makeDepRules)
	RegisterDataType("raw_deps.v0", []*dep2.RawDependency{})
	RegisterDataType("resolved_deps.v0", []*dep2.ResolvedDep{})
}

func makeDepRules(c *config.Repository, commitID string, existing []makefile.Rule) ([]makefile.Rule, error) {
	if len(c.SourceUnits) == 0 {
		return nil, nil
	}

	resolveRule := &ResolveDepsRule{DataFileInfo{c.URI, commitID, reflect.TypeOf([]*dep2.ResolvedDep{})}, nil}

	rules := []makefile.Rule{resolveRule}
	for _, u := range c.SourceUnits {
		rule := &ListSourceUnitDepsRule{DataFileInfo{c.URI, commitID, reflect.TypeOf([]*dep2.RawDependency{})}, u}
		rules = append(rules, rule)
		resolveRule.rawDepLists = append(resolveRule.rawDepLists, rule.Target())
	}

	return rules, nil
}

type ResolveDepsRule struct {
	targetInfo  DataFileInfo
	rawDepLists []makefile.Prereq
}

func (r *ResolveDepsRule) Target() makefile.Target {
	return &RepositoryCommitDataFile{r.targetInfo}
}

func (r *ResolveDepsRule) Prereqs() []makefile.Prereq { return r.rawDepLists }

func (r *ResolveDepsRule) Recipes() []string {
	return []string{"srcgraph -v resolve-deps -json $^ 1> $@"}
}

type ListSourceUnitDepsRule struct {
	targetInfo DataFileInfo
	unit       unit.SourceUnit
}

func (r *ListSourceUnitDepsRule) Target() makefile.Target {
	return &SourceUnitDataFile{r.targetInfo, r.unit}
}

func (r *ListSourceUnitDepsRule) Prereqs() []makefile.Prereq {
	return makefile.FilePrereqs(r.unit.Paths())
}

func (r *ListSourceUnitDepsRule) Recipes() []string {
	return []string{fmt.Sprintf("srcgraph -v list-deps -json %q 1> $@", unit.MakeID(r.unit))}
}
