//+build off

// TMP reenable this

package vcsutil

import (
	"fmt"
	"path/filepath"

	"github.com/sourcegraph/makex"

	"github.com/sourcegraph/srclib/buildstore"
	"github.com/sourcegraph/srclib/config"
	"github.com/sourcegraph/srclib/plan"
	"github.com/sourcegraph/srclib/unit"
)

func init() {
	plan.RegisterRuleMaker("blame", makeBlameRules)
	buildstore.RegisterDataType("blame.v0", &BlameOutput{})
}

func makeBlameRules(c *config.Repository, dataDir string, existing []makex.Rule) ([]makex.Rule, error) {
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

func (r *BlameSourceUnitRule) Prereqs() []string { return r.Unit.Files }

func (r *BlameSourceUnitRule) Recipes() []string {
	return []string{
		"mkdir -p `dirname $@`",
		fmt.Sprintf("srcgraph blame %s 1> $@", makex.Quote(string(r.Unit.ID()))),
	}
}
