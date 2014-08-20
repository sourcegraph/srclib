package vcsutil

import (
	"fmt"
	"path/filepath"

	"github.com/sourcegraph/makex"

	"sourcegraph.com/sourcegraph/srclib/buildstore"
	"sourcegraph.com/sourcegraph/srclib/config"
	"sourcegraph.com/sourcegraph/srclib/plan"
	"sourcegraph.com/sourcegraph/srclib/unit"
)

func init() {
	plan.RegisterRuleMaker("blame", makeBlameRules)
	buildstore.RegisterDataType("blame", &BlameOutput{})
}

func makeBlameRules(c *config.Tree, dataDir string, existing []makex.Rule, opt plan.Options) ([]makex.Rule, error) {
	// blame each source unit individually
	var rules []makex.Rule
	for _, u := range c.SourceUnits {
		rules = append(rules, &BlameSourceUnitRule{dataDir, u})
	}
	return rules, nil
}

type BlameSourceUnitRule struct {
	dataDir string
	Unit    *unit.SourceUnit
}

func (r *BlameSourceUnitRule) Target() string {
	return filepath.Join(r.dataDir, plan.SourceUnitDataFilename(&BlameOutput{}, r.Unit))
}

func (r *BlameSourceUnitRule) Prereqs() []string {
	ps := []string{filepath.Join(r.dataDir, plan.SourceUnitDataFilename(unit.SourceUnit{}, r.Unit))}
	ps = append(ps, r.Unit.Files...)
	return ps
}

func (r *BlameSourceUnitRule) Recipes() []string {
	return []string{
		fmt.Sprintf("src internal unit-blame --unit-data %s 1> $@", makex.Quote(filepath.Join(r.dataDir, plan.SourceUnitDataFilename(unit.SourceUnit{}, r.Unit)))),
	}
}

func (r *BlameSourceUnitRule) SourceUnit() *unit.SourceUnit { return r.Unit }
