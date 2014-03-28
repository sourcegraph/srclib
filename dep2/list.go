package dep2

import (
	"encoding/json"
	"reflect"
	"runtime"
	"sync"

	"code.google.com/p/rog-go/parallel"
	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/container"
	"sourcegraph.com/sourcegraph/srcgraph/task2"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
)

// Listers maps source unit struct types to registered dependency listers.
var Listers = make(map[reflect.Type][]Lister)

// RegisterLister adds a dependency lister for the given source unit type. If
// Register is called twice with the same source unit type, the second
// dependency lister is added to the list associated with the type.
func RegisterLister(emptySourceUnit unit.SourceUnit, lister Lister) {
	typ := ptrTo(emptySourceUnit)
	if lister == nil {
		panic("dep2: Register lister is nil")
	}
	Listers[typ] = append(Listers[typ], lister)
}

func ptrTo(v interface{}) reflect.Type {
	typ := reflect.TypeOf(v)
	if typ.Kind() != reflect.Ptr {
		typ = reflect.PtrTo(typ)
	}
	return typ
}

// RawDependency represents a declaration of a dependency.
type RawDependency struct {
	// FromUnit is the source unit name in which this dependency is declared.
	FromUnit string

	// FromUnitType is the source unit type in which this dependency is declared.
	FromUnitType string

	// FromFile is the file in which the dependency is declared. If empty, it is
	// assumed that the declaration can't be traced to a specific file (or that
	// such tracing has not been implemented yet).
	//
	// For example, FromFile is typically a "package.json" file for NPM packages,
	// because that's where dependencies are declared.
	FromFile string `json:",omitempty"`

	// FromStart is the character offset in FromFile where the
	// dependency declaration begins, or 0 if the position is not known.
	FromStart int `json:",omitempty"`

	// FromEnd is the character offset in FromFile where the dependency
	// declaration ends. If both FromStart and FromEnd are 0, then it is assumed
	// that no character range information is known.
	FromEnd int `json:",omitempty"`

	// TargetType is a string describing what kind of dependency this is. This
	// string corresponds to the target type passed to RegisterResolver.
	TargetType string

	// Target stores custom data that identifies the dependency.
	//
	// For example, Target is typically a Go import path string for Go packages.
	// For NPM packages, Target contains the key-value pair in the package.json
	// file's "dependencies" object, specifying the dependency's NPM package
	// name and version (or source URL).
	Target interface{}
}

type Lister interface {
	List(dir string, unit unit.SourceUnit, c *config.Repository, x *task2.Context) ([]*RawDependency, error)
}

type ListerBuilder interface {
	BuildLister(dir string, unit unit.SourceUnit, c *config.Repository, x *task2.Context) (*container.Command, error)
}

type DockerLister struct {
	ListerBuilder
}

func (l DockerLister) List(dir string, unit unit.SourceUnit, c *config.Repository, x *task2.Context) ([]*RawDependency, error) {
	cmd, err := l.BuildLister(dir, unit, c, x)
	if err != nil {
		return nil, err
	}

	data, err := cmd.Run()
	if err != nil {
		return nil, err
	}

	var deps []*RawDependency
	err = json.Unmarshal(data, &deps)
	if err != nil {
		return nil, err
	}

	return deps, nil
}

// List lists all dependencies of the source unit (whose repository is cloned to
// dir), using all registered Listers.
func List(dir string, u unit.SourceUnit, c *config.Repository, x *task2.Context) ([]*RawDependency, error) {
	var deps struct {
		list []*RawDependency
		sync.Mutex
	}
	deps.list = make([]*RawDependency, 0)
	run := parallel.NewRun(runtime.GOMAXPROCS(0))
	for _, l_ := range Listers[ptrTo(u)] {
		l := l_
		run.Do(func() error {
			deps2, err := l.List(dir, u, c, x)
			if err != nil {
				return err
			}

			for _, d := range deps2 {
				d.FromUnit, d.FromUnitType = u.Name(), unit.Type(u)
			}

			deps.Lock()
			defer deps.Unlock()
			deps.list = append(deps.list, deps2...)
			return nil
		})
	}
	err := run.Wait()
	return deps.list, err
}
