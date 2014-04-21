package build

import (
	"fmt"
	"path/filepath"

	"github.com/sourcegraph/makex"
	"sourcegraph.com/sourcegraph/srcgraph/buildstore"
	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/grapher2"
	_ "sourcegraph.com/sourcegraph/srcgraph/toolchain/all_toolchains"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
)

func init() {
	RegisterRuleMaker("graph", makeGraphRules)
	buildstore.RegisterDataType("graph.v0", &grapher2.Output{})
}

func makeGraphRules(c *config.Repository, dataDir string, existing []makex.Rule) ([]makex.Rule, error) {
	var rules []makex.Rule
	for _, u := range c.SourceUnits {
		rules = append(rules, &GraphUnitRule{dataDir, u})
	}
	return rules, nil
}

type GraphUnitRule struct {
	dataDir string
	Unit    unit.SourceUnit
}

func (r *GraphUnitRule) Target() string {
	return filepath.Join(r.dataDir, SourceUnitDataFilename(&grapher2.Output{}, r.Unit))
}

func (r *GraphUnitRule) Prereqs() []string { return r.Unit.Paths() }

func (r *GraphUnitRule) Recipes() []string {
	return []string{
		"mkdir -p `dirname $@`",
		fmt.Sprintf("srcgraph -v graph -json %q 1> $@", unit.MakeID(r.Unit)),
	}
}
