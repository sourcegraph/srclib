package build

import (
	"fmt"
	"path/filepath"

	"sourcegraph.com/sourcegraph/repo"
	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/scan"
	"sourcegraph.com/sourcegraph/srcgraph/task2"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
	"sourcegraph.com/sourcegraph/srcgraph/util2/makefile"
)

type RuleMaker func(c *config.Repository, commitID string) ([]makefile.Rule, error)

var RuleMakers = make(map[string]RuleMaker)

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
}

func CreateMakefile(dir, cloneURL, commitID string, x *task2.Context) ([]makefile.Rule, error) {
	repoURI := repo.MakeURI(cloneURL)
	c, err := scan.ReadDirConfigAndScan(dir, repoURI, x)
	if err != nil {
		return nil, err
	}

	var allRules []makefile.Rule
	for name, r := range RuleMakers {
		rules, err := r(c, commitID)
		if err != nil {
			return nil, fmt.Errorf("rule maker %s: %s", name, err)
		}
		allRules = append(allRules, rules...)
	}

	return allRules, nil
}

type Target interface {
	makefile.Target
	RelName() string
}
type RepositoryCommitSpec struct {
	RepositoryURI repo.URI
	CommitID      string
}

type RepositoryCommitOutputFile struct {
	RepositoryCommitSpec
	Suffix string
}

func (f *RepositoryCommitOutputFile) Name() string {
	return filepath.Join(WorkDir, string(f.RepositoryURI), f.CommitID, f.RelName())
}

func (f *RepositoryCommitOutputFile) RelName() string {
	return f.Suffix + ".json"
}

type SourceUnitSpec struct {
	RepositoryURI repo.URI
	CommitID      string
	Unit          unit.SourceUnit
}

type SourceUnitOutputFile struct {
	SourceUnitSpec
	Suffix string
}

func (f *SourceUnitOutputFile) Name() string {
	return filepath.Join(WorkDir, string(f.RepositoryURI), f.CommitID, f.RelName())
}

func (f *SourceUnitOutputFile) RelName() string {
	return filepath.Clean(fmt.Sprintf("%s_%s.json", unit.MakeID(f.Unit), f.Suffix))
}
