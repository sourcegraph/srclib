package dep

import (
	"fmt"
	"path/filepath"

	"sourcegraph.com/sourcegraph/makex"
	"sourcegraph.com/sourcegraph/srclib"
	"sourcegraph.com/sourcegraph/srclib/buildstore"
	"sourcegraph.com/sourcegraph/srclib/config"
	"sourcegraph.com/sourcegraph/srclib/graph2"
	"sourcegraph.com/sourcegraph/srclib/plan"
	"sourcegraph.com/sourcegraph/srclib/toolchain"
	"sourcegraph.com/sourcegraph/srclib/util"
)

func init() {
	plan.RegisterRuleMaker2(depresolveOp, makeDepRules2)
	buildstore.RegisterDataType("depresolve2", []*graph2.Dep{})
}

func makeDepRules2(c *config.Tree2, dataDir string, existing []makex.Rule) ([]makex.Rule, error) {
	const op = depresolveOp
	var rules []makex.Rule
	for _, u := range c.Units {
		var toolRef *srclib.ToolRef = nil
		// toolRef := u.Ops[op]
		if toolRef == nil {
			choice, err := toolchain.ChooseTool(depresolveOp, u.UnitType)
			if err != nil {
				return nil, err
			}
			toolRef = choice
		}

		if toolRef != nil {
			rules = append(rules, &ResolveDepsRule2{dataDir, u, toolRef})
		}
	}
	return rules, nil
}

type ResolveDepsRule2 struct {
	dataDir string
	Unit    *graph2.Unit
	Tool    *srclib.ToolRef
}

func (r *ResolveDepsRule2) Target() string {
	return filepath.ToSlash(filepath.Join(r.dataDir, plan.SourceUnitDataFilename2([]*graph2.Dep{}, r.Unit)))
}

func (r *ResolveDepsRule2) Prereqs() []string {
	return []string{filepath.ToSlash(filepath.Join(r.dataDir, plan.SourceUnitDataFilename2(graph2.Unit{}, r.Unit)))}
}

func (r *ResolveDepsRule2) Recipes() []string {
	if r.Tool == nil {
		return nil
	}
	return []string{
		fmt.Sprintf("%s tool %q %q < $^ 1> $@", util.SafeCommandName(srclib.CommandName), r.Tool.Toolchain, r.Tool.Subcmd+"2"),
	}
}
