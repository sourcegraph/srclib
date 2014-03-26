package build

import (
	"fmt"
	"reflect"

	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/grapher2"
	_ "sourcegraph.com/sourcegraph/srcgraph/toolchain/all_toolchains"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
	"sourcegraph.com/sourcegraph/srcgraph/util2/makefile"
)

func init() {
	RegisterRuleMaker("graph", makeGraphRules)
	RegisterDataType("graph.v0", grapher2.Output{})
}

func makeGraphRules(c *config.Repository, commitID string, existing []makefile.Rule) ([]makefile.Rule, error) {
	var rules []makefile.Rule
	for _, u := range c.SourceUnits {
		rules = append(rules, &GraphSourceUnitRule{DataFileInfo{c.URI, commitID, reflect.TypeOf(grapher2.Output{})}, u})
	}
	return rules, nil
}

type GraphSourceUnitRule struct {
	targetInfo DataFileInfo
	Unit       unit.SourceUnit
}

func (r *GraphSourceUnitRule) Target() makefile.Target {
	return &SourceUnitDataFile{r.targetInfo, r.Unit}
}

func (r *GraphSourceUnitRule) Prereqs() []makefile.Prereq { return makefile.FilePrereqs(r.Unit.Paths()) }

func (r *GraphSourceUnitRule) Recipes() []string {
	return []string{fmt.Sprintf("srcgraph -v graph -json %q 1> $@", unit.MakeID(r.Unit))}
}
