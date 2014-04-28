package authorship

import (
	"fmt"
	"path/filepath"

	"github.com/sourcegraph/makex"
	"sourcegraph.com/sourcegraph/srcgraph/build"
	"sourcegraph.com/sourcegraph/srcgraph/buildstore"
	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/graph"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
	"sourcegraph.com/sourcegraph/srcgraph/vcsutil"
)

func init() {
	build.RegisterRuleMaker("authorship", makeAuthorshipRules)
	buildstore.RegisterDataType("unit-authorship.v0", &SourceUnitOutput{})
}

type SourceUnitOutput struct {
	Symbols             map[graph.SymbolPath][]*SymbolAuthorship
	Refs                []*RefAuthorship
	Authors             []*AuthorStats
	ClientsOfOtherUnits []*ClientStats
}

// makeAuthorshipRules makes rules for computing authorship information about
// symbols and refs at a source unit level and a repository level.
func makeAuthorshipRules(c *config.Repository, dataDir string, existing []makex.Rule) ([]makex.Rule, error) {
	// determine authorship for each source unit individually, but we have to
	// wait until graphing AND blaming completes.
	graphRules, blameRules := make(map[unit.ID]*build.GraphUnitRule), make(map[unit.ID]*vcsutil.BlameSourceUnitRule)
	for _, rule := range existing {
		switch rule := rule.(type) {
		case *build.GraphUnitRule:
			graphRules[unit.MakeID(rule.Unit)] = rule
		case *vcsutil.BlameSourceUnitRule:
			blameRules[unit.MakeID(rule.Unit)] = rule
		}
	}

	var rules []makex.Rule
	for unitID, gr := range graphRules {
		// find unit
		var u unit.SourceUnit
		for _, u2 := range c.SourceUnits {
			if unit.MakeID(u2) == unitID {
				u = u2
				break
			}
		}
		if u == nil {
			return nil, fmt.Errorf("no source unit found with ID %q", unitID)
		}

		br, present := blameRules[unitID]
		if !present {
			return nil, fmt.Errorf("no blame rule found corresponding to graph rule for unit ID %q", unit.MakeID(u))
		}

		rule := &ComputeUnitAuthorshipRule{
			dataDir:     dataDir,
			Unit:        u,
			BlameOutput: br.Target(),
			GraphOutput: gr.Target(),
		}

		rules = append(rules, rule)
	}

	return rules, nil
}

type ComputeUnitAuthorshipRule struct {
	dataDir     string
	Unit        unit.SourceUnit
	BlameOutput string
	GraphOutput string
}

func (r *ComputeUnitAuthorshipRule) Target() string {
	return filepath.Join(r.dataDir, build.SourceUnitDataFilename(&SourceUnitOutput{}, r.Unit))
}

func (r *ComputeUnitAuthorshipRule) Prereqs() []string { return []string{r.BlameOutput, r.GraphOutput} }

func (r *ComputeUnitAuthorshipRule) Recipes() []string {
	return []string{
		"mkdir -p `dirname $@`",
		fmt.Sprintf("srcgraph authorship %s %s 1> $@", makex.Quote(r.BlameOutput), makex.Quote(r.GraphOutput)),
	}
}
