package dep2

import (
	"fmt"

	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/task2"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
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
	Resolve(dep *RawDependency, c *config.Repository, x *task2.Context) (*ResolvedTarget, error)
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

	// ToUnitID is the ID of the source unit in ToRepo that is depended on.
	ToUnitID unit.ID

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
func Resolve(dep *RawDependency, c *config.Repository, x *task2.Context) (*ResolvedTarget, error) {
	r, registered := Resolvers[dep.TargetType]
	if !registered {
		return nil, fmt.Errorf("no resolver registered for raw dependency target type %q", dep.TargetType)
	}

	return r.Resolve(dep, c, x)
}

func ResolveAll(rawDeps []*RawDependency, c *config.Repository, x *task2.Context) ([]*ResolvedDep, error) {
	var resolved []*ResolvedDep
	for _, rawDep := range rawDeps {
		rt, err := Resolve(rawDep, c, x)
		if err != nil {
			return nil, err
		}
		rd := &ResolvedDep{
			FromRepo:       c.URI,
			FromUnitID:     rawDep.DefUnitID,
			ResolvedTarget: rt,
		}
		resolved = append(resolved, rd)
	}
	return resolved, nil
}
