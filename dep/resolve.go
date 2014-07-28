package dep

import (
	"path/filepath"

	"github.com/sourcegraph/makex"
	"github.com/sourcegraph/srclib/buildstore"
	"github.com/sourcegraph/srclib/config"
	"github.com/sourcegraph/srclib/plan"
	"github.com/sourcegraph/srclib/toolchain"
	"github.com/sourcegraph/srclib/unit"
)

func init() {
	plan.RegisterRuleMaker("depresolve", makeDepRules)
	buildstore.RegisterDataType("depresolve.v0", []*ResolvedDep{})
}

func makeDepRules(c *config.Repository, dataDir string, existing []makex.Rule) ([]makex.Rule, error) {
	if len(c.SourceUnits) == 0 {
		return nil, nil
	}

	var rules []makex.Rule
	for _, u := range c.SourceUnits {
		rules = append(rules, &ResolveDepsRule{dataDir, u})
	}

	return rules, nil
}

type ResolveDepsRule struct {
	dataDir string
	Unit    *unit.SourceUnit
}

func (r *ResolveDepsRule) Target() string {
	return filepath.Join(r.dataDir, plan.SourceUnitDataFilename([]*ResolvedDep{}, r.Unit))
}

func (r *ResolveDepsRule) Prereqs() []string {
	return []string{filepath.Join(r.dataDir, plan.RepositoryCommitDataFilename(&config.Repository{}))}
}

func (r *ResolveDepsRule) Recipes() []string {
	return []string{"src tool github.com/sourcegraph/srclib-go depresolve < $^ 1> $@"}
}

// ResolvedTarget represents a resolved dependency target.
type ResolvedTarget struct {
	// ToRepoCloneURL is the clone URL of the repository that is depended on.
	//
	// When graphers emit ResolvedDependencies, they should fill in this field,
	// not ToRepo, so that the dependent repository can be added if it doesn't
	// exist. The ToRepo URI alone does not specify enough information to add
	// the repository (because it doesn't specify the VCS type, scheme, etc.).
	ToRepoCloneURL string

	// ToUnit is the name of the source unit that is depended on.
	ToUnit string

	// ToUnitType is the type of the source unit that is depended on.
	ToUnitType string

	// ToVersion is the version of the dependent repository (if known),
	// according to whatever version string specifier is used by FromRepo's
	// dependency management system.
	ToVersionString string

	// ToRevSpec specifies the desired VCS revision of the dependent repository
	// (if known).
	ToRevSpec string
}

// Resolution is the result of dependency resolution: either a successfully
// resolved target or an error.
type Resolution struct {
	// Raw is the original raw dep that this was resolution was attempted on.
	Raw interface{}

	// Target is the resolved dependency, if resolution succeeds.
	Target *ResolvedTarget `json:",omitempty"`

	// Error is the resolution error, if any.
	Error string `json:",omitempty"`
}

// Command for dep resolution has no options.
type Command struct{}

// ResolveDeps resolves dependencies
func ResolveDeps(resolver toolchain.Tool, cmd Command, unit *unit.SourceUnit) ([]*Resolution, error) {
	args, err := toolchain.MarshalArgs(&cmd)
	if err != nil {
		return nil, err
	}

	var res []*Resolution
	if err := resolver.Run(args, unit, &res); err != nil {
		return nil, err
	}

	return res, nil
}

// func ResolveAll(rawDeps []*RawDependency, c *config.Repository) ([]*ResolvedDep, error) {
// 	var resolved []*ResolvedDep
// 	for _, rawDep := range rawDeps {
// 		rt, err := Resolve(rawDep, c)
// 		if err != nil {
// 			return nil, err
// 		}
// 		if rt == nil {
// 			continue
// 		}

// 		var toRepo repo.URI
// 		if rt.ToRepoCloneURL == "" {
// 			// empty clone URL means the current repository
// 			toRepo = c.URI
// 		} else {
// 			toRepo = repo.MakeURI(rt.ToRepoCloneURL)
// 		}

// 		// TODO!(sqs): return repo clone URLs as well, so we can add new repositories
// 		rd := &ResolvedDep{
// 			FromRepo:        c.URI,
// 			FromUnit:        rawDep.FromUnit,
// 			FromUnitType:    rawDep.FromUnitType,
// 			ToRepo:          toRepo,
// 			ToUnit:          rt.ToUnit,
// 			ToUnitType:      rt.ToUnitType,
// 			ToVersionString: rt.ToVersionString,
// 			ToRevSpec:       rt.ToRevSpec,
// 		}
// 		resolved = append(resolved, rd)
// 	}
// 	sort.Sort(resolvedDeps(resolved))
// 	return resolved, nil
// }

// type resolvedDeps []*ResolvedDep

// func (d *ResolvedDep) sortKey() string    { b, _ := json.Marshal(d); return string(b) }
// func (l resolvedDeps) Len() int           { return len(l) }
// func (l resolvedDeps) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }
// func (l resolvedDeps) Less(i, j int) bool { return l[i].sortKey() < l[j].sortKey() }
