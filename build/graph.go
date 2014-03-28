package build

import (
	"fmt"
	"path/filepath"
	"reflect"

	"sourcegraph.com/sourcegraph/srcgraph/buildstore"
	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/grapher2"
	_ "sourcegraph.com/sourcegraph/srcgraph/toolchain/all_toolchains"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
	"sourcegraph.com/sourcegraph/srcgraph/util2/makefile"
)

func init() {
	RegisterRuleMaker("graph", makeGraphRules)
	buildstore.RegisterDataType("graph.v0", grapher2.Output{})
}

func makeGraphRules(c *config.Repository, dataDir string, existing []makefile.Rule) ([]makefile.Rule, error) {
	var rules []makefile.Rule
	for _, u := range c.SourceUnits {
		rules = append(rules, &GraphSourceUnitRule{reflect.TypeOf(grapher2.Output{}), dataDir, u})
	}
	return rules, nil
}

type GraphSourceUnitRule struct {
	targetDataType reflect.Type
	dataDir        string
	Unit           unit.SourceUnit
}

func (r *GraphSourceUnitRule) Target() string {
	return filepath.Join(r.dataDir, SourceUnitDataFilename(r.targetDataType, r.Unit))
}

func (r *GraphSourceUnitRule) Prereqs() []string { return r.Unit.Paths() }

func (r *GraphSourceUnitRule) Recipes() []string {
	return []string{fmt.Sprintf("srcgraph -v graph -json %q 1> $@", unit.MakeID(r.Unit))}
}
