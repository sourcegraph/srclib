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
	Unit(unit.Key) (*unit.SourceUnit, error)

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

// allUnits is a UnitFilter that selects all units.
func allUnits(*unit.SourceUnit) bool { return true }

func unitKeyFilter(key unit.Key) UnitFilter {
	return func(unit *unit.SourceUnit) bool {
		return unit.Type == key.UnitType && unit.Name == key.Unit
	}
}

func defUnitFilter(key unit.Key) DefFilter {
	return func(def *graph.Def) bool {
		return key.UnitType == def.UnitType && key.Unit == def.Unit
	}
}

func refUnitFilter(key unit.Key) RefFilter {
	return func(ref *graph.Ref) bool {
		return key.UnitType == ref.UnitType && key.Unit == ref.Unit
	}
}

// A multiTreeStore is a TreeStore whose methods call the
// corresponding method on each of the tree stores returned by the
// treeStores func.
type multiTreeStore struct {
	treeStores func() (map[string]TreeStore, error)
}

var _ TreeStore = (*multiTreeStore)(nil)

func (s multiTreeStore) Unit(key unit.Key) (*unit.SourceUnit, error) {
	tss, err := s.treeStores()
	if err != nil {
		return nil, err
	}

	for commitID, ts := range tss {
		if key.CommitID != commitID {
			continue
		}
		unit, err := ts.Unit(key)
		if err != nil {
			if IsNotExist(err) {
				continue
			}
			return nil, err
		}
		return unit, nil
	}
	return nil, errUnitNotExist
}

func (s multiTreeStore) Units(f UnitFilter) ([]*unit.SourceUnit, error) {
	if f == nil {
		f = allUnits
	}

	tss, err := s.treeStores()
	if err != nil {
		return nil, err
	}

	var allUnits []*unit.SourceUnit
	for commitID, ts := range tss {
		units, err := ts.Units(func(unit *unit.SourceUnit) bool {
			unit.CommitID = commitID
			return f(unit)
		})
		if err != nil {
			return nil, err
		}
		allUnits = append(allUnits, units...)
	}
	return allUnits, nil
}

func (s multiTreeStore) Def(key graph.DefKey) (*graph.Def, error) {
	tss, err := s.treeStores()
	if err != nil {
		return nil, err
	}

	for commitID, ts := range tss {
		if key.CommitID != commitID {
			continue
		}
		def, err := ts.Def(key)
		if err != nil {
			if IsNotExist(err) {
				continue
			}
			return nil, err
		}
		def.CommitID = commitID
		return def, nil
	}
	return nil, errDefNotExist
}

func (s multiTreeStore) Defs(f DefFilter) ([]*graph.Def, error) {
	if f == nil {
		f = allDefs
	}

	tss, err := s.treeStores()
	if err != nil {
		return nil, err
	}

	var allDefs []*graph.Def
	for commitID, ts := range tss {
		defs, err := ts.Defs(func(def *graph.Def) bool {
			def.CommitID = commitID
			return f(def)
		})
		if err != nil {
			return nil, err
		}
		allDefs = append(allDefs, defs...)
	}
	return allDefs, nil
}

func (s multiTreeStore) Refs(f RefFilter) ([]*graph.Ref, error) {
	if f == nil {
		f = allRefs
	}

	tss, err := s.treeStores()
	if err != nil {
		return nil, err
	}

	var allRefs []*graph.Ref
	for commitID, ts := range tss {
		refs, err := ts.Refs(func(ref *graph.Ref) bool {
			ref.CommitID = commitID
			return f(ref)
		})
		if err != nil {
			return nil, err
		}
		allRefs = append(allRefs, refs...)
	}
	return allRefs, nil
}
