package build

import (
	"fmt"
	"path/filepath"

	"sourcegraph.com/sourcegraph/repo"
	"sourcegraph.com/sourcegraph/srcgraph/buildstore"
	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/scan"
	"sourcegraph.com/sourcegraph/srcgraph/task2"
	"sourcegraph.com/sourcegraph/srcgraph/util2/makefile"
)

type RuleMaker func(c *config.Repository, dataDir string, existing []makefile.Rule) ([]makefile.Rule, error)

var (
	RuleMakers        = make(map[string]RuleMaker)
	ruleMakerNames    []string
	orderedRuleMakers []RuleMaker
)

// RegisterRuleMaker adds a function that creates a list of build rules for a
// repository. If RegisterRuleMaker is called twice with the same target or
// target name, if name or r are nil, it panics.
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

func CreateMakefile(dir, cloneURL, commitID string, x *task2.Context) ([]byte, error) {
	repoURI := repo.MakeURI(cloneURL)
	c, err := scan.ReadDirConfigAndScan(dir, repoURI, x)
	if err != nil {
		return nil, err
	}

	repoStore, err := buildstore.NewRepositoryStore(dir)
	if err != nil {
		return nil, err
	}

	rootDataDir, err := buildstore.RootDir(repoStore)
	if err != nil {
		return nil, err
	}
	dataDir, err := filepath.Rel(dir, filepath.Join(rootDataDir, repoStore.CommitPath(commitID)))
	if err != nil {
		return nil, err
	}

	var allRules []makefile.Rule
	for i, r := range orderedRuleMakers {
		name := ruleMakerNames[i]
		rules, err := r(c, dataDir, allRules)
		if err != nil {
			return nil, fmt.Errorf("rule maker %s: %s", name, err)
		}
		allRules = append(allRules, rules...)
	}

	header := []string{
		fmt.Sprintf("_ = $(shell mkdir -p %s)", makefile.Quote(dataDir)),

		// DELETE_ON_ERROR makes it so that the targets for failed recipes are
		// deleted. This lets us do "1> $@" to write to the target file without
		// erroneously satisfying the target if the recipe fails.
		".DELETE_ON_ERROR:",
	}

	mf, err := makefile.Makefile(allRules, header)
	if err != nil {
		return nil, err
	}
	return mf, nil
}
