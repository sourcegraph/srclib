package authorship

import (
	"fmt"
	"path/filepath"

	"github.com/sourcegraph/makex"
	"sourcegraph.com/sourcegraph/srclib/buildstore"
	"sourcegraph.com/sourcegraph/srclib/config"
	"sourcegraph.com/sourcegraph/srclib/graph"
	"sourcegraph.com/sourcegraph/srclib/grapher"
	"sourcegraph.com/sourcegraph/srclib/plan"
	"sourcegraph.com/sourcegraph/srclib/unit"
	"sourcegraph.com/sourcegraph/srclib/vcsutil"
)

func init() {
	plan.RegisterRuleMaker("authorship", makeAuthorshipRules)
	buildstore.RegisterDataType("authorship", &SourceUnitOutput{})
}

type SourceUnitOutput struct {
	Defs                map[graph.DefPath][]*DefAuthorship
	Refs                []*RefAuthorship
	Authors             []*AuthorStats
	ClientsOfOtherUnits []*ClientStats
}

// makeAuthorshipRules makes rules for computing authorship information about
// defs and refs at a source unit level and a repository level.
func makeAuthorshipRules(c *config.Tree, dataDir string, existing []makex.Rule, opt plan.Options) ([]makex.Rule, error) {
	// determine authorship for each source unit individually, but we have to
	// wait until graphing AND blaming completes.
	graphRules, blameRules := make(map[unit.ID]*grapher.GraphUnitRule), make(map[unit.ID]*vcsutil.BlameSourceUnitRule)
	for _, rule := range existing {
		switch rule := rule.(type) {
		case *grapher.GraphUnitRule:
			graphRules[rule.Unit.ID()] = rule
		case *vcsutil.BlameSourceUnitRule:
			blameRules[rule.Unit.ID()] = rule
		}
	}

	var rules []makex.Rule
	for unitID, gr := range graphRules {
		// find unit
		var u *unit.SourceUnit
		for _, u2 := range c.SourceUnits {
			if u2.ID() == unitID {
				u = u2
				break
			}
		}
		if u == nil {
			return nil, fmt.Errorf("no source unit found with ID %q", unitID)
		}

		br, present := blameRules[unitID]
		if !present {
			return nil, fmt.Errorf("no blame rule found corresponding to graph rule for unit ID %q", u.ID())
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
	Unit        *unit.SourceUnit
	BlameOutput string
	GraphOutput string
}

func (r *ComputeUnitAuthorshipRule) Target() string {
	return filepath.Join(r.dataDir, plan.SourceUnitDataFilename(&SourceUnitOutput{}, r.Unit))
}

func (r *ComputeUnitAuthorshipRule) Prereqs() []string { return []string{r.BlameOutput, r.GraphOutput} }

func (r *ComputeUnitAuthorshipRule) Recipes() []string {
	return []string{
		fmt.Sprintf("src internal unit-authorship --blame-data %s --graph-data %s 1> $@", makex.Quote(r.BlameOutput), makex.Quote(r.GraphOutput)),
	}
}

func (r *ComputeUnitAuthorshipRule) SourceUnit() *unit.SourceUnit { return r.Unit }
