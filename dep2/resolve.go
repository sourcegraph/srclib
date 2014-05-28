package dep2

import (
	"encoding/json"
	"fmt"

	"sort"

	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/container"
	"sourcegraph.com/sourcegraph/srcgraph/repo"
)

// Resolvers maps RawDependency.TargetType strings to their registered
// RawDependency.Target resolver.
var Resolvers = make(map[string]Resolver)

// Register adds a dependency resolver for the given RawDependency.TargetType.
// If Register is called twice with the same type, or if resolver is nil, it
// panics.
func RegisterResolver(targetType string, resolver Resolver) {
	if _, registered := Resolvers[targetType]; registered {
		panic("dep2: RegisterResolver called twice for target type " + targetType)
	}
	if resolver == nil {
		panic("dep2: RegisterResolver resolver is nil")
	}
	Resolvers[targetType] = resolver
}

type Resolver interface {
	Resolve(dep *RawDependency, c *config.Repository) (*ResolvedTarget, error)
}

type ResolverBuilder interface {
	BuildResolver(dep *RawDependency, c *config.Repository) (*container.Command, error)
}

type DockerResolver struct {
	ResolverBuilder
}

func (l DockerResolver) Resolve(dep *RawDependency, c *config.Repository) (*ResolvedTarget, error) {
	cmd, err := l.BuildResolver(dep, c)
	if err != nil {
		return nil, err
	}
	if cmd == nil {
		return nil, nil
	}

	data, err := cmd.Run()
	if err != nil {
		return nil, err
	}

	var resolv *ResolvedTarget
	err = json.Unmarshal(data, &resolv)
	if err != nil {
		return nil, err
	}

	return resolv, nil
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

// Resolve resolves a raw dependency using the registered resolver for the
// RawDependency's TargetType.
func Resolve(dep *RawDependency, c *config.Repository) (*ResolvedTarget, error) {
	r, registered := Resolvers[dep.TargetType]
	if !registered {
		return nil, fmt.Errorf("no resolver registered for raw dependency target type %q", dep.TargetType)
	}

	return r.Resolve(dep, c)
}

func ResolveAll(rawDeps []*RawDependency, c *config.Repository) ([]*ResolvedDep, error) {
	var resolved []*ResolvedDep
	for _, rawDep := range rawDeps {
		rt, err := Resolve(rawDep, c)
		if err != nil {
			return nil, err
		}
		if rt == nil {
			continue
		}

		var toRepo repo.URI
		if rt.ToRepoCloneURL == "" {
			// empty clone URL means the current repository
			toRepo = c.URI
		} else {
			toRepo = repo.MakeURI(rt.ToRepoCloneURL)
		}

		// TODO!(sqs): return repo clone URLs as well, so we can add new repositories
		rd := &ResolvedDep{
			FromRepo:        c.URI,
			FromUnit:        rawDep.FromUnit,
			FromUnitType:    rawDep.FromUnitType,
			ToRepo:          toRepo,
			ToUnit:          rt.ToUnit,
			ToUnitType:      rt.ToUnitType,
			ToVersionString: rt.ToVersionString,
			ToRevSpec:       rt.ToRevSpec,
		}
		resolved = append(resolved, rd)
	}
	sort.Sort(resolvedDeps(resolved))
	return resolved, nil
}

type resolvedDeps []*ResolvedDep

func (d *ResolvedDep) sortKey() string    { b, _ := json.Marshal(d); return string(b) }
func (l resolvedDeps) Len() int           { return len(l) }
func (l resolvedDeps) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }
func (l resolvedDeps) Less(i, j int) bool { return l[i].sortKey() < l[j].sortKey() }
