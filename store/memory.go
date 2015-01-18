package store

import (
	"errors"

	"sourcegraph.com/sourcegraph/srclib/graph"
)

// A memoryUnitStore is a UnitStore that stores data in flat files
// in a filesystem.
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
