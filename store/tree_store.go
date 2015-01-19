package store

import (
	"sourcegraph.com/sourcegraph/srclib/graph"
	"sourcegraph.com/sourcegraph/srclib/unit"
)

// A TreeStore stores and accesses srclib build data for an arbitrary
// source tree (consisting of any number of source units).
type TreeStore interface {
	// Unit gets a single unit by its unit type and name. If no such
	// unit exists, an error satisfying IsNotExist is returned.
	Unit(typ, name string) (*unit.SourceUnit, error)

	// Units returns all units that match the filter.
	Units(UnitFilter) ([]*unit.SourceUnit, error)

	// UnitStore's methods call the corresponding methods on the
	// UnitStore of each source unit contained within this tree. The
	// combined results are returned (in undefined order).
	UnitStore
}

// A TreeImporter imports srclib build data for a source unit into a
// TreeStore.
type TreeImporter interface {
	// Import imports a source unit and its graph data into the
	// store. If Import is called with a nil SourceUnit and output
	// data, the importer considers the tree to have no source units
	// until others are imported in the future (this makes it possible
	// to distinguish between a tree that has no source units and a
	// tree whose source units simply haven't been imported yet).
	Import(*unit.SourceUnit, graph.Output) error
}

// A UnitFilter is used to filter a list of units to only those for
// which the func returns true.
type UnitFilter func(*unit.SourceUnit) bool

func allUnits(*unit.SourceUnit) bool { return true }

// unitKey is the key for a source unit within a tree.
type unitKey struct{ typ, name string }

func unitKeyFilter(key unitKey) UnitFilter {
	return func(unit *unit.SourceUnit) bool {
		return unit.Type == key.typ && unit.Name == key.name
	}
}

func defUnitFilter(key unitKey) DefFilter {
	return func(def *graph.Def) bool {
		return key.typ == def.UnitType && key.name == def.Unit
	}
}

func refUnitFilter(key unitKey) RefFilter {
	return func(ref *graph.Ref) bool {
		return key.typ == ref.UnitType && key.name == ref.Unit
	}
}
