package plan

import (
	"fmt"
	"sort"

	"sourcegraph.com/sourcegraph/makex"
	"sourcegraph.com/sourcegraph/srclib/buildstore"
	"sourcegraph.com/sourcegraph/srclib/config"
)

type RuleMaker func(c *config.Tree, dataDir string, existing []makex.Rule) ([]makex.Rule, error)

var (
	RuleMakers        = make(map[string]RuleMaker)
	ruleMakerNames    []string
	orderedRuleMakers []RuleMaker
)

// RegisterRuleMaker adds a function that creates a list of build rules for a
// repository. If RegisterRuleMaker is called twice with the same target or
// target name, if name is empty, or if r is nil, it panics.
func RegisterRuleMaker(name string, r RuleMaker) {
	if _, dup := RuleMakers[name]; dup {
		panic("build: Register called twice for target lister " + name)
	}
	if r == nil {
		panic("build: Register target is nil")
	}
	RuleMakers[name] = r
	ruleMakerNames = append(ruleMakerNames, name)
	orderedRuleMakers = append(orderedRuleMakers, r)
}

// CreateMakefile creates the makefiles for the source units in c.
func CreateMakefile(buildDataDir string, buildStore buildstore.RepoBuildStore, vcsType string, c *config.Tree) (*makex.Makefile, error) {
	var allRules []makex.Rule
	for i, r := range orderedRuleMakers {
		name := ruleMakerNames[i]
		rules, err := r(c, buildDataDir, allRules)
		if err != nil {
			return nil, fmt.Errorf("rule maker %s: %s", name, err)
		}
		sort.Sort(ruleSort{rules})
		allRules = append(allRules, rules...)
	}

	// Add an "all" rule at the very beginning.
	allTargets := make([]string, len(allRules))
	for i, rule := range allRules {
		allTargets[i] = rule.Target()
	}
	allRule := &makex.BasicRule{TargetFile: "all", PrereqFiles: allTargets}
	allRules = append([]makex.Rule{allRule}, allRules...)

	// DELETE_ON_ERROR makes it so that the targets for failed recipes are
	// deleted. This lets us do "1> $@" to write to the target file without
	// erroneously satisfying the target if the recipe fails. makex has this
	// behavior by default and does not heed .DELETE_ON_ERROR.
	allRules = append(allRules, &makex.BasicRule{TargetFile: ".DELETE_ON_ERROR"})

	mf := &makex.Makefile{Rules: allRules}

	return mf, nil
}

type RuleMaker2 func(c *config.Tree2, dataDir string, existing []makex.Rule) ([]makex.Rule, error)

var (
	RuleMakers2        = make(map[string]RuleMaker2)
	ruleMakerNames2    []string
	orderedRuleMakers2 []RuleMaker2
)

// RegisterRuleMaker adds a function that creates a list of build rules for a
// repository. If RegisterRuleMaker is called twice with the same target or
// target name, if name is empty, or if r is nil, it panics.
func RegisterRuleMaker2(name string, r RuleMaker2) {
	if _, dup := RuleMakers2[name]; dup {
		panic("build: Register called twice for target lister " + name)
	}
	if r == nil {
		panic("build: Register target is nil")
	}
	RuleMakers2[name] = r
	ruleMakerNames2 = append(ruleMakerNames2, name)
	orderedRuleMakers2 = append(orderedRuleMakers2, r)
}

// CreateMakefile creates the makefiles for the source units in c.
func CreateMakefile2(buildDataDir string, buildStore buildstore.RepoBuildStore, vcsType string, c *config.Tree2) (*makex.Makefile, error) {
	var allRules []makex.Rule
	for i, r := range orderedRuleMakers2 {
		name := ruleMakerNames2[i]
		rules, err := r(c, buildDataDir, allRules)
		if err != nil {
			return nil, fmt.Errorf("rule maker %s: %s", name, err)
		}
		sort.Sort(ruleSort{rules})
		allRules = append(allRules, rules...)
	}

	// Add an "all" rule at the very beginning.
	allTargets := make([]string, len(allRules))
	for i, rule := range allRules {
		allTargets[i] = rule.Target()
	}
	allRule := &makex.BasicRule{TargetFile: "all", PrereqFiles: allTargets}
	allRules = append([]makex.Rule{allRule}, allRules...)

	// DELETE_ON_ERROR makes it so that the targets for failed recipes are
	// deleted. This lets us do "1> $@" to write to the target file without
	// erroneously satisfying the target if the recipe fails. makex has this
	// behavior by default and does not heed .DELETE_ON_ERROR.
	allRules = append(allRules, &makex.BasicRule{TargetFile: ".DELETE_ON_ERROR"})

	mf := &makex.Makefile{Rules: allRules}

	return mf, nil
}
