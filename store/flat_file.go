package store

import (
	"encoding/json"

	"sourcegraph.com/sourcegraph/rwvfs"
	"sourcegraph.com/sourcegraph/srclib/graph"
)

// A flatFileUnitStore is a UnitStore that stores data in flat files
// in a filesystem.
type flatFileUnitStore struct {
	fs rwvfs.FileSystem
}

const flatFileName = "data.json"

func (s *flatFileUnitStore) Def(key graph.DefKey) (*graph.Def, error) {
	defs, err := s.Defs(defKeyFilter(key))
	if err != nil {
		return nil, err
	}
	if len(defs) == 0 {
		return nil, errDefNotExist
	}
	return defs[0], nil
}

func (s *flatFileUnitStore) Defs(f DefFilter) ([]*graph.Def, error) {
	o, err := s.open()
	if err != nil {
		return nil, err
	}

	if f == nil {
		f = allDefs
	}
	var defs []*graph.Def
	for _, def := range o.Defs {
		if f(def) {
			defs = append(defs, def)
		}
	}
	return defs, nil
}

func (s *flatFileUnitStore) Refs(f RefFilter) ([]*graph.Ref, error) {
	o, err := s.open()
	if err != nil {
		return nil, err
	}

	if f == nil {
		f = allRefs
	}
	var refs []*graph.Ref
	for _, ref := range o.Refs {
		if f(ref) {
			refs = append(refs, ref)
		}
	}
	return refs, nil
}

func (s *flatFileUnitStore) Import(data graph.Output) (err error) {
	f, err := s.fs.Create(flatFileName)
	if err != nil {
		return err
	}
	defer func() {
		err2 := f.Close()
		if err == nil {
			err = err2
		}
	}()
	return json.NewEncoder(f).Encode(data)
}

func (s *flatFileUnitStore) open() (o *graph.Output, err error) {
	f, err := s.fs.Open(flatFileName)
	if err != nil {
		return nil, err
	}
	defer func() {
		err2 := f.Close()
		if err == nil {
			err = err2
		}
	}()
	o = &graph.Output{}
	return o, json.NewDecoder(f).Decode(o)
}

func (s *flatFileUnitStore) String() string { return "flatFileUnitStore" }
