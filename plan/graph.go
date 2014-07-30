//+build off

package plan

import (
	"fmt"
	"path/filepath"

	"github.com/sourcegraph/makex"
	"sourcegraph.com/sourcegraph/srclib/buildstore"
	"sourcegraph.com/sourcegraph/srclib/config"
	"sourcegraph.com/sourcegraph/srclib/grapher"
	"sourcegraph.com/sourcegraph/srclib/unit"
)

func init() {
	RegisterRuleMaker("graph", makeGraphRules)
	buildstore.RegisterDataType("graph.v0", &grapher.Output{})
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
	Unit    *unit.SourceUnit
}

func (r *GraphUnitRule) Target() string {
	return filepath.Join(r.dataDir, SourceUnitDataFilename(&grapher.Output{}, r.Unit))
}

func (r *GraphUnitRule) Prereqs() []string { return r.Unit.Files }

func (r *GraphUnitRule) Recipes() []string {
	return []string{
		"mkdir -p `dirname $@`",
		fmt.Sprintf("srcgraph -v graph -json %q 1> $@", r.Unit.ID()),
	}
}
