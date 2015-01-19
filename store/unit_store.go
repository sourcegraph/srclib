package store

import (
	"sourcegraph.com/sourcegraph/srclib/graph"
	"sourcegraph.com/sourcegraph/srclib/unit"
)

// A UnitStore stores and accesses srclib build data for a single
// source unit.
type UnitStore interface {
	// Def gets a single def by its key. If no such def exists, an
	// error satisfying IsNotExist is returned.
	Def(graph.DefKey) (*graph.Def, error)

	// Defs returns all defs that match the filter.
	Defs(DefFilter) ([]*graph.Def, error)

	// Refs returns all refs that match the filter.
	Refs(RefFilter) ([]*graph.Ref, error)

	// TODO(sqs): how to deal with depresolve and other non-graph
	// data?
}

// A UnitImporter imports srclib build data for a single source unit
// into a UnitStore.
type UnitImporter interface {
	// Import imports defs, refs, etc., into the store. It overwrites
	// all existing data for this source unit (and at the commit, if
	// applicable).
	Import(graph.Output) error
}

// A DefFilter is used to filter a list of defs to only those for
// which the func returns true.
type DefFilter func(*graph.Def) bool

// allDefs is a DefFilter that selects all defs.
func allDefs(*graph.Def) bool { return true }

func defKeyFilter(key graph.DefKey) DefFilter {
	return func(def *graph.Def) bool {
		return def.DefKey == key
	}
}

func defPathFilter(path string) DefFilter {
	return func(def *graph.Def) bool {
		return def.Path == path
	}
}

// A RefFilter is used to filter a list of refs to only those for
// which the func returns true.
type RefFilter func(*graph.Ref) bool

// allRefs is a RefFilter that selects all refs.
func allRefs(*graph.Ref) bool { return true }

// A multiUnitStore is a UnitStore whose methods call the
// corresponding method on each of the unit stores returned by the
// unitStores func.
type multiUnitStore struct {
	unitStores func() (map[unit.Key]UnitStore, error)
}

var _ UnitStore = (*multiUnitStore)(nil)

func (s multiUnitStore) Def(key graph.DefKey) (*graph.Def, error) {
	uss, err := s.unitStores()
	if err != nil {
		return nil, err
	}

	for unitKey, us := range uss {
		if !defUnitFilter(unitKey)(&graph.Def{DefKey: key}) {
			continue
		}
		def, err := us.Def(key)
		if err != nil {
			if IsNotExist(err) {
				continue
			}
			return nil, err
		}
		if def.UnitType == "" {
			def.UnitType = unitKey.UnitType
		}
		if def.Unit == "" {
			def.Unit = unitKey.Unit
		}
		return def, nil
	}
	return nil, errDefNotExist
}

func (s multiUnitStore) Defs(f DefFilter) ([]*graph.Def, error) {
	if f == nil {
		f = allDefs
	}

	uss, err := s.unitStores()
	if err != nil {
		return nil, err
	}

	var allDefs []*graph.Def
	for unitKey, us := range uss {
		defs, err := us.Defs(func(def *graph.Def) bool {
			if def.UnitType == "" {
				def.UnitType = unitKey.UnitType
			}
			if def.Unit == "" {
				def.Unit = unitKey.Unit
			}
			return f(def)
		})
		if err != nil {
			return nil, err
		}
		allDefs = append(allDefs, defs...)
	}
	return allDefs, nil
}

func (s multiUnitStore) Refs(f RefFilter) ([]*graph.Ref, error) {
	if f == nil {
		f = allRefs
	}

	uss, err := s.unitStores()
	if err != nil {
		return nil, err
	}

	var allRefs []*graph.Ref
	for unitKey, us := range uss {
		refs, err := us.Refs(func(ref *graph.Ref) bool {
			if ref.UnitType == "" {
				ref.UnitType = unitKey.UnitType
			}
			if ref.Unit == "" {
				ref.Unit = unitKey.Unit
			}
			return f(ref)
		})
		if err != nil {
			return nil, err
		}
		allRefs = append(allRefs, refs...)
	}
	return allRefs, nil
}
