package build

import (
	"fmt"
	"path/filepath"
	"reflect"

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

	resolveRule := &ResolveDepsRule{reflect.TypeOf([]*dep2.ResolvedDep{}), dataDir, nil}

	rules := []makefile.Rule{resolveRule}
	for _, u := range c.SourceUnits {
		rule := &ListSourceUnitDepsRule{reflect.TypeOf([]*dep2.RawDependency{}), dataDir, u}
		rules = append(rules, rule)
		resolveRule.rawDepLists = append(resolveRule.rawDepLists, rule.Target())
	}

	return rules, nil
}

type ResolveDepsRule struct {
	targetDataType reflect.Type
	dataDir        string
	rawDepLists    []string
}

func (r *ResolveDepsRule) Target() string {
	return filepath.Join(r.dataDir, RepositoryCommitDataFilename(r.targetDataType))
}

func (r *ResolveDepsRule) Prereqs() []string { return r.rawDepLists }

func (r *ResolveDepsRule) Recipes() []string {
	return []string{"srcgraph -v resolve-deps -json $^ 1> $@"}
}

type ListSourceUnitDepsRule struct {
	targetDataType reflect.Type
	dataDir        string
	unit           unit.SourceUnit
}

func (r *ListSourceUnitDepsRule) Target() string {
	return filepath.Join(r.dataDir, SourceUnitDataFilename(r.targetDataType, r.unit))
}

func (r *ListSourceUnitDepsRule) Prereqs() []string {
	return r.unit.Paths()
}

func (r *ListSourceUnitDepsRule) Recipes() []string {
	return []string{fmt.Sprintf("srcgraph -v list-deps -json %q 1> $@", unit.MakeID(r.unit))}
}
