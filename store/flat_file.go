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

// A flatFileRepoStore is a RepoStore that stores data in flatFile.
type flatFileRepoStore struct {
	fs rwvfs.FileSystem
	multiTreeStore
}

func newFlatFileRepoStore(fs rwvfs.FileSystem) *flatFileRepoStore {
	ts := &flatFileRepoStore{fs: fs}
	ts.multiTreeStore = multiTreeStore{treeStores: ts.treeStores}
	return ts
}

func (s *flatFileRepoStore) Version(commitID string) (*Version, error) {
	versions, err := s.Versions(versionCommitIDFilter(commitID))
	if err != nil {
		return nil, err
	}
	if len(versions) == 0 {
		return nil, errVersionNotExist
	}
	return versions[0], nil
}

func (s *flatFileRepoStore) Versions(f VersionFilter) ([]*Version, error) {
	if f == nil {
		f = allVersions
	}

	versionDirs, err := s.versionDirs()
	if err != nil {
		return nil, err
	}

	var versions []*Version
	for _, dir := range versionDirs {
		version := &Version{CommitID: path.Base(dir)}
		if f(version) {
			versions = append(versions, version)
		}
	}
	return versions, nil
}

func (s *flatFileRepoStore) versionDirs() ([]string, error) {
	entries, err := s.fs.ReadDir(".")
	if err != nil {
		return nil, err
	}
	dirs := make([]string, len(entries))
	for i, e := range entries {
		dirs[i] = e.Name()
	}
	return dirs, nil
}

func (s *flatFileRepoStore) Import(commitID string, unit *unit.SourceUnit, data graph.Output) error {
	ts := newFlatFileTreeStore(rwvfs.Sub(s.fs, commitID))
	return ts.Import(unit, data)
}

func (s *flatFileRepoStore) treeStores() (map[string]TreeStore, error) {
	versionDirs, err := s.versionDirs()
	if err != nil {
		return nil, err
	}

	tss := make(map[string]TreeStore, len(versionDirs))
	for _, dir := range versionDirs {
		commitID := path.Base(dir)
		tss[commitID] = newFlatFileTreeStore(rwvfs.Sub(s.fs, commitID))
	}
	return tss, nil
}

func (s *flatFileRepoStore) String() string { return "flatFileRepoStore" }

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

func (s *flatFileTreeStore) Unit(key unit.Key) (*unit.SourceUnit, error) {
	units, err := s.Units(unitKeyFilter(key))
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

func (s *flatFileTreeStore) unitFilename(key unit.Key) string {
	return path.Join(key.UnitType, key.Unit+unitFileSuffix)
}

const unitFileSuffix = ".unit.json"

func (s *flatFileTreeStore) Import(unit *unit.SourceUnit, data graph.Output) (err error) {
	if unit == nil {
		return rwvfs.MkdirAll(s.fs, ".")
	}

	filename := s.unitFilename(unit.Key())
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

	dir := strings.TrimSuffix(s.unitFilename(unit.Key()), unitFileSuffix)
	if err := rwvfs.MkdirAll(s.fs, dir); err != nil {
		return err
	}
	us := &flatFileUnitStore{fs: rwvfs.Sub(s.fs, dir)}
	return us.Import(data)
}

func (s *flatFileTreeStore) unitStores() (map[unit.Key]UnitStore, error) {
	unitFiles, err := s.unitFilenames()
	if err != nil {
		return nil, err
	}

	uss := make(map[unit.Key]UnitStore, len(unitFiles))
	for _, unitFile := range unitFiles {
		dir := strings.TrimSuffix(unitFile, unitFileSuffix)
		unitKey := unit.Key{UnitType: path.Dir(dir), Unit: path.Base(dir)}
		uss[unitKey] = &flatFileUnitStore{fs: rwvfs.Sub(s.fs, dir)}
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
