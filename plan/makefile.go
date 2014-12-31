package plan

import (
	"fmt"
	"log"

	"strings"

	"sourcegraph.com/sourcegraph/makex"
	"sourcegraph.com/sourcegraph/srclib/config"
	"sourcegraph.com/sourcegraph/srclib/unit"
)

type Options struct {
	ToolchainExecOpt string
}

type RuleMaker func(c *config.Tree, dataDir string, existing []makex.Rule, opt Options) ([]makex.Rule, error)

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

// SAMER: needs a name other than 'cached'.
type cachedRule struct {
	cachedPath string
	target     string
	unit       *unit.SourceUnit
	prereqs    []string
}

func (r *cachedRule) Target() string {
	return r.target
}

func (r *cachedRule) Prereqs() []string {
	return r.prereqs
}

func (r *cachedRule) Recipes() []string {
	return []string{
		// SAMER: will this blow up if s/cp/ln -s/ ?
		fmt.Sprintf("cp %s %s", r.cachedPath, r.target),
	}
}

// SAMER: do I need this?
func (r *cachedRule) SourceUnit() *unit.SourceUnit {
	return r.unit
}

// CreateMakeFile creats the make files for the source units in c.
// buildDataDir has the format "[dataDir, e.g. '.srclib-cache']/[vcs hash]"
func CreateMakefile(buildDataDir string, c *config.Tree, opt Options) (*makex.Makefile, error) {
	var allRules []makex.Rule
	for i, r := range orderedRuleMakers {
		name := ruleMakerNames[i]
		rules, err := r(c, buildDataDir, allRules, opt)
		if err != nil {
			return nil, fmt.Errorf("rule maker %s: %s", name, err)
		}
		// SAMER: check option flag first.
		// Replace rules for cached source units
		for i, rule := range rules {
			r, ok := rule.(interface {
				SourceUnit() *unit.SourceUnit
			})
			if !ok {
				continue
			}
			u := r.SourceUnit()
			if u.CachedRev == "" {
				continue
			}
			t := strings.Split(rule.Target(), "/")
			// Sanity checks
			if len(t) < 3 ||
				strings.Join(t[0:2], "/") != buildDataDir ||
				// Mercurial and Git both use 40-char hashes.
				len(t[1]) != 40 {
				// TODO(samer): how can we reliably check that the file exists?
				log.Println(buildDataDir, t, t[0], t[1])
				log.Printf("Aborted caching of %s because target is in the wrong format!", rule.Target())
				continue
			}
			t[1] = u.CachedRev

			rules[i] = &cachedRule{
				cachedPath: strings.Join(t, "/"),
				target:     rule.Target(),
				unit:       u,
				prereqs:    rule.Prereqs(),
			}
		}
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
