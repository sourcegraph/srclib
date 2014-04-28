package vcsutil

import (
	"fmt"
	"path/filepath"

	"github.com/sourcegraph/makex"

	"sourcegraph.com/sourcegraph/srcgraph/build"
	"sourcegraph.com/sourcegraph/srcgraph/buildstore"
	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
)

func init() {
	build.RegisterRuleMaker("blame", makeBlameRules)
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
	Unit    unit.SourceUnit
}

func (r *BlameSourceUnitRule) Target() string {
	return filepath.Join(r.dataDir, build.SourceUnitDataFilename(&BlameOutput{}, r.Unit))
}

func (r *BlameSourceUnitRule) Prereqs() []string { return r.Unit.Paths() }

func (r *BlameSourceUnitRule) Recipes() []string {
	return []string{
		"mkdir -p `dirname $@`",
		fmt.Sprintf("srcgraph blame %s 1> $@", makex.Quote(string(unit.MakeID(r.Unit)))),
	}
}
