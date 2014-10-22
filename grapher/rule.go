package grapher

import (
	"fmt"
	"path/filepath"

	"github.com/sourcegraph/makex"
	"sourcegraph.com/sourcegraph/srclib/buildstore"
	"sourcegraph.com/sourcegraph/srclib/config"
	"sourcegraph.com/sourcegraph/srclib/plan"
	"sourcegraph.com/sourcegraph/srclib/toolchain"
	"sourcegraph.com/sourcegraph/srclib/unit"
)

const graphOp = "graph"

func init() {
	plan.RegisterRuleMaker(graphOp, makeGraphRules)
	buildstore.RegisterDataType("graph", &Output{})
}

func makeGraphRules(c *config.Tree, dataDir string, existing []makex.Rule, opt plan.Options) ([]makex.Rule, error) {
	const op = graphOp
	var rules []makex.Rule
	for _, u := range c.SourceUnits {
		toolRef := u.Ops[op]
		if toolRef == nil {
			choice, err := toolchain.ChooseTool(graphOp, u.Type)
			if err != nil {
				return nil, err
			}
			toolRef = choice
		}

		rules = append(rules, &GraphUnitRule{dataDir, u, toolRef, opt})
	}
	return rules, nil
}

type GraphUnitRule struct {
	dataDir string
	Unit    *unit.SourceUnit
	Tool    *toolchain.ToolRef
	opt     plan.Options
}

func (r *GraphUnitRule) Target() string {
	return filepath.Join(r.dataDir, plan.SourceUnitDataFilename(&Output{}, r.Unit))
}

func (r *GraphUnitRule) Prereqs() []string {
	ps := []string{filepath.Join(r.dataDir, plan.SourceUnitDataFilename(unit.SourceUnit{}, r.Unit))}
	ps = append(ps, r.Unit.Files...)
	return ps
}

func (r *GraphUnitRule) Recipes() []string {
	return []string{
		fmt.Sprintf("src tool %s %q %q < $< | src internal normalize-graph-data --unit-type %q --dir . 1> $@", r.opt.ToolchainExecOpt, r.Tool.Toolchain, r.Tool.Subcmd, r.Unit.Type),
	}
}

func (r *GraphUnitRule) SourceUnit() *unit.SourceUnit { return r.Unit }
