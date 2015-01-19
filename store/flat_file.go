package store

import (
	"encoding/json"
	"os"
	"path"

	"github.com/kr/fs"

	"strings"

	"sourcegraph.com/sourcegraph/rwvfs"
	"sourcegraph.com/sourcegraph/srclib/graph"
	"sourcegraph.com/sourcegraph/srclib/unit"
)

// A flatFileTreeStore is a TreeStore that stores data in flat files
// in a filesystem.
type flatFileTreeStore struct {
	fs rwvfs.FileSystem
	multiUnitStore
}

func newFlatFileTreeStore(fs rwvfs.FileSystem) *flatFileTreeStore {
	ts := &flatFileTreeStore{fs: fs}
	ts.multiUnitStore = multiUnitStore{ts.unitStores}
	return ts
}

func (s *flatFileTreeStore) Unit(typ, name string) (*unit.SourceUnit, error) {
	units, err := s.Units(unitKeyFilter(unitKey{typ, name}))
	if err != nil {
		return nil, err
	}
	if len(units) == 0 {
		return nil, errUnitNotExist
	}
	return units[0], nil
}

func (s *flatFileTreeStore) Units(f UnitFilter) ([]*unit.SourceUnit, error) {
	if f == nil {
		f = allUnits
	}

	unitFilenames, err := s.unitFilenames()
	if err != nil {
		return nil, err
	}

	var units []*unit.SourceUnit
	for _, filename := range unitFilenames {
		unit, err := s.openUnitFile(filename)
		if err != nil {
			return nil, err
		}
		if f(unit) {
			units = append(units, unit)
		}
	}
	return units, nil
}

func (s *flatFileTreeStore) openUnitFile(filename string) (*unit.SourceUnit, error) {
	f, err := s.fs.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errUnitNotExist
		}
		return nil, err
	}
	defer func() {
		err2 := f.Close()
		if err == nil {
			err = err2
		}
	}()

	var unit unit.SourceUnit
	return &unit, json.NewDecoder(f).Decode(&unit)
}

func (s *flatFileTreeStore) unitFilenames() ([]string, error) {
	var files []string
	w := fs.WalkFS(".", rwvfs.Walkable(s.fs))
	for w.Step() {
		if err := w.Err(); err != nil {
			return nil, err
		}
		fi := w.Stat()
		if fi.Mode().IsRegular() && strings.HasSuffix(fi.Name(), unitFileSuffix) {
			files = append(files, w.Path())
		}
	}
	return files, nil
}

func (s *flatFileTreeStore) unitFilename(typ, name string) string {
	return path.Join(typ, name+unitFileSuffix)
}

const unitFileSuffix = ".unit.json"

func (s *flatFileTreeStore) Import(unit *unit.SourceUnit, data graph.Output) (err error) {
	if unit == nil {
		return rwvfs.MkdirAll(s.fs, ".")
	}

	filename := s.unitFilename(unit.Type, unit.Name)
	if err := rwvfs.MkdirAll(s.fs, path.Dir(filename)); err != nil {
		return err
	}
	f, err := s.fs.Create(filename)
	if err != nil {
		return err
	}
	defer func() {
		err2 := f.Close()
		if err == nil {
			err = err2
		}
	}()
	if err := json.NewEncoder(f).Encode(unit); err != nil {
		return err
	}

	dir := strings.TrimSuffix(s.unitFilename(unit.Type, unit.Name), unitFileSuffix)
	if err := rwvfs.MkdirAll(s.fs, dir); err != nil {
		return err
	}
	us := &flatFileUnitStore{fs: rwvfs.Sub(s.fs, dir)}
	return us.Import(data)
}

func (s *flatFileTreeStore) unitStores() (map[unitKey]UnitStore, error) {
	unitFiles, err := s.unitFilenames()
	if err != nil {
		return nil, err
	}

	uss := make(map[unitKey]UnitStore, len(unitFiles))
	for _, unitFile := range unitFiles {
		dir := strings.TrimSuffix(unitFile, unitFileSuffix)
		typ, name := path.Dir(dir), path.Base(dir)
		uss[unitKey{typ, name}] = &flatFileUnitStore{fs: rwvfs.Sub(s.fs, dir)}
	}
	return uss, nil
}

func (s *flatFileTreeStore) String() string { return "flatFileTreeStore" }

// A flatFileUnitStore is a UnitStore that stores data in flat files
// in a filesystem.
type flatFileUnitStore struct {
	fs rwvfs.FileSystem
}

const flatFileUnitDataFilename = "data.json"

func (s *flatFileUnitStore) Def(key graph.DefKey) (*graph.Def, error) {
	defs, err := s.Defs(defPathFilter(key.Path))
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
	f, err := s.fs.Create(flatFileUnitDataFilename)
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
	f, err := s.fs.Open(flatFileUnitDataFilename)
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
