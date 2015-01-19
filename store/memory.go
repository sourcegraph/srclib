package store

import (
	"errors"

	"sourcegraph.com/sourcegraph/srclib/graph"
	"sourcegraph.com/sourcegraph/srclib/unit"
)

// A memoryTreeStore is a TreeStore that stores data in memory.
type memoryTreeStore struct {
	units []*unit.SourceUnit
	data  map[unitKey]*graph.Output
	multiUnitStore
}

func newMemoryTreeStore() *memoryTreeStore {
	ts := &memoryTreeStore{}
	ts.multiUnitStore = multiUnitStore{unitStores: ts.unitStores}
	return ts
}

var errTreeNoInit = errors.New("tree not yet initialized")

func (s *memoryTreeStore) Unit(typ, name string) (*unit.SourceUnit, error) {
	if s.units == nil {
		return nil, errTreeNoInit
	}

	for _, unit := range s.units {
		if unit.Type == typ && unit.Name == name {
			return unit, nil
		}
	}
	return nil, errUnitNotExist
}

func (s *memoryTreeStore) Units(f UnitFilter) ([]*unit.SourceUnit, error) {
	if f == nil {
		f = allUnits
	}

	if s.units == nil {
		return nil, errTreeNoInit
	}

	var units []*unit.SourceUnit
	for _, unit := range s.units {
		if f(unit) {
			units = append(units, unit)
		}

	}
	return units, nil
}

func (s *memoryTreeStore) Import(u *unit.SourceUnit, data graph.Output) error {
	if s.units == nil {
		s.units = []*unit.SourceUnit{}
	}
	if s.data == nil {
		s.data = map[unitKey]*graph.Output{}
	}
	if u == nil {
		return nil
	}

	s.units = append(s.units, u)
	s.data[unitKey{u.Type, u.Name}] = &data
	return nil
}

func (s *memoryTreeStore) unitStores() (map[unitKey]UnitStore, error) {
	if s.data == nil {
		return nil, errTreeNoInit
	}

	uss := make(map[unitKey]UnitStore, len(s.data))
	for unitKey, data := range s.data {
		uss[unitKey] = &memoryUnitStore{data: data}
	}
	return uss, nil
}

func (s *memoryTreeStore) String() string { return "memoryTreeStore" }

// A memoryUnitStore is a UnitStore that stores data in memory.
type memoryUnitStore struct {
	data *graph.Output
}

var errNoDataImported = errors.New("memory store: no data imported")

func (s *memoryUnitStore) Def(key graph.DefKey) (*graph.Def, error) {
	if s.data == nil {
		return nil, errNoDataImported
	}

	for _, def := range s.data.Defs {
		if def.Path == key.Path {
			return def, nil
		}
	}
	return nil, errDefNotExist
}

func (s *memoryUnitStore) Defs(f DefFilter) ([]*graph.Def, error) {
	if s.data == nil {
		return nil, errNoDataImported
	}

	if f == nil {
		f = allDefs
	}
	var defs []*graph.Def
	for _, def := range s.data.Defs {
		if f(def) {
			defs = append(defs, def)
		}
	}
	return defs, nil
}

func (s *memoryUnitStore) Refs(f RefFilter) ([]*graph.Ref, error) {
	if s.data == nil {
		return nil, errNoDataImported
	}

	if f == nil {
		f = allRefs
	}
	var refs []*graph.Ref
	for _, ref := range s.data.Refs {
		if f(ref) {
			refs = append(refs, ref)
		}
	}
	return refs, nil
}

func (s *memoryUnitStore) Import(data graph.Output) error {
	s.data = &data
	return nil
}

func (s *memoryUnitStore) String() string { return "memoryUnitStore" }
