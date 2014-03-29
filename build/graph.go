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
		rules = append(rules, &GraphSourceUnitRule{dataDir, u})
	}
	return rules, nil
}

type GraphSourceUnitRule struct {
	dataDir string
	Unit    unit.SourceUnit
}

func (r *GraphSourceUnitRule) Target() string {
	return filepath.Join(r.dataDir, SourceUnitDataFilename(&grapher2.Output{}, r.Unit))
}

func (r *GraphSourceUnitRule) Prereqs() []string { return r.Unit.Paths() }

func (r *GraphSourceUnitRule) Recipes() []string {
	return []string{
		"mkdir -p `dirname $@`",
		fmt.Sprintf("srcgraph -v graph -json %q 1> $@", unit.MakeID(r.Unit)),
	}
}
