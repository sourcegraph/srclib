package build

import (
	"fmt"
	"path/filepath"
	"strings"

	"sourcegraph.com/sourcegraph/repo"
	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/scan"
	"sourcegraph.com/sourcegraph/srcgraph/task2"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
	"sourcegraph.com/sourcegraph/srcgraph/util2/makefile"
)

type RuleMaker func(c *config.Repository, existing []makefile.Rule) ([]makefile.Rule, error)

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

func CreateMakefile(dir, cloneURL, commitID string, x *task2.Context) ([]makefile.Rule, []string, error) {
	repoURI := repo.MakeURI(cloneURL)
	c, err := scan.ReadDirConfigAndScan(dir, repoURI, x)
	if err != nil {
		return nil, nil, err
	}

	var allRules []makefile.Rule
	for i, r := range orderedRuleMakers {
		name := ruleMakerNames[i]
		rules, err := r(c, allRules)
		if err != nil {
			return nil, nil, fmt.Errorf("rule maker %s: %s", name, err)
		}
		allRules = append(allRules, rules...)
	}

	vars := []string{
		fmt.Sprintf("outdir = %s", makefile.Quote(filepath.Join(WorkDir, string(repoURI), commitID))),
		"_ = $(shell mkdir -p ${outdir})",
	}

	return allRules, vars, nil
}

func SubstituteVars(s string, vars []string) string {
	for _, v := range vars {
		p := strings.SplitN(v, "=", 2)
		name, val := strings.TrimSuffix(p[0], " "), strings.TrimPrefix(p[1], " ")
		s = strings.Replace(s, "${"+name+"}", val, -1)
	}
	return s
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
	Suffix string
}

func (f *RepositoryCommitOutputFile) Name() string {
	return filepath.Join("${outdir}", f.RelName())
}

func (f *RepositoryCommitOutputFile) RelName() string {
	return f.Suffix + ".json"
}

type SourceUnitSpec struct {
	Unit unit.SourceUnit
}

type SourceUnitOutputFile struct {
	Unit   unit.SourceUnit
	Suffix string
}

func (f *SourceUnitOutputFile) Name() string {
	return filepath.Join("${outdir}", f.RelName())
}

func (f *SourceUnitOutputFile) RelName() string {
	return filepath.Clean(fmt.Sprintf("%s_%s.json", unit.MakeID(f.Unit), f.Suffix))
}
