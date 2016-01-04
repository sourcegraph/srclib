package grapher

import (
	"fmt"
	"os"
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
	plan.RegisterRuleMaker2(graphOp, makeGraphRules2)
	buildstore.RegisterDataType("graph2", &graph2.Output{})
}

func makeGraphRules2(c *config.Tree2, dataDir string, existing []makex.Rule) ([]makex.Rule, error) {
	const op = graphOp
	var rules []makex.Rule
	for _, u := range c.Units {
		var toolRef *srclib.ToolRef = nil
		// toolRef := u.Ops[op]
		if toolRef == nil {
			choice, err := toolchain.ChooseTool(graphOp, u.UnitType)
			if err != nil {
				return nil, err
			}
			toolRef = choice
		}

		if toolRef != nil {
			rules = append(rules, &GraphUnitRule2{dataDir, u, toolRef})
		}
	}
	return rules, nil
}

type GraphUnitRule2 struct {
	dataDir string
	Unit    *graph2.Unit
	Tool    *srclib.ToolRef
}

var _ makex.Rule = (*GraphUnitRule2)(nil)

func (r *GraphUnitRule2) Target() string {
	return filepath.ToSlash(filepath.Join(r.dataDir, plan.SourceUnitDataFilename2(&graph2.Output{}, r.Unit)))
}

func (r *GraphUnitRule2) Prereqs() []string {
	ps := []string{filepath.ToSlash(filepath.Join(r.dataDir, plan.SourceUnitDataFilename2(graph2.Unit{}, r.Unit)))}
	for _, file := range r.Unit.Files {
		if _, err := os.Stat(file); err != nil && os.IsNotExist(err) {
			// skip not-existent files listed in source unit
			continue
		}
		ps = append(ps, file)
	}
	return ps
}

func (r *GraphUnitRule2) Recipes() []string {
	if r.Tool == nil {
		return nil
	}
	safeCommand := util.SafeCommandName(srclib.CommandName)
	return []string{
		// fmt.Sprintf(`%s tool %q %q < $< | %s internal normalize-graph-data --unit-type %q --dir . 1> $@`, safeCommand, r.Tool.Toolchain, r.Tool.Subcmd+"2", safeCommand, r.Unit.UnitType),

		// TODO: add back in normalization
		fmt.Sprintf(`%s tool %q %q < $< 1> $@`, safeCommand, r.Tool.Toolchain, r.Tool.Subcmd+"2"),
	}
}
