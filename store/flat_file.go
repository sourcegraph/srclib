package store

import (
	"os"
	"path"
	"strings"

	"github.com/kr/fs"

	"sourcegraph.com/sourcegraph/rwvfs"
	"sourcegraph.com/sourcegraph/srclib/graph"
	"sourcegraph.com/sourcegraph/srclib/unit"
)

// FlatFileConfig configures a flat file store.
type FlatFileConfig struct {
	Codec Codec
}

const SrclibStoreDir = ".srclib-store"

// A flatFileMultiRepoStore is a MultiRepoStore that stores data in
// flat files.
type flatFileMultiRepoStore struct {
	fs rwvfs.FileSystem
	repoStores
	codec Codec
}

// NewFlatFileMultiRepoStore creates a new repository store (that can
// be imported into) that is backed by files on a filesystem.
//
// The repoPathFunc takes a repo ID (URI) and returns the
// slash-delimited subpath where its data should be stored in the
// multi-repo store. If nil, it defaults to a function that returns
// the string passed in plus "/.srclib-store".
//
// The isRepoPathFunc takes a subpath and returns true if it is a repo
// path returned by repoPathFunc. This is used when listing all repos
// (in the Repos method). Repos may not be nested, so if
// isRepoPathFunc returns true for a dir, it is not recursively
// searched for repos. If nil, it defaults to returning true if the
// last path component is ".srclib-store" (which means it works with
// the default repoPathFunc).
func NewFlatFileMultiRepoStore(fs rwvfs.FileSystem, conf *FlatFileConfig) MultiRepoStoreImporter {
	if conf == nil {
		conf = &FlatFileConfig{}
	}
	if conf.Codec == nil {
		conf.Codec = JSONCodec{}
	}

	setCreateParentDirs(fs)
	mrs := &flatFileMultiRepoStore{fs: fs, codec: conf.Codec}
	mrs.repoStores = repoStores{repoStores: mrs.getRepoStores}
	return mrs
}

func (s *flatFileMultiRepoStore) Repo(id string) (string, error) {
	repos, err := s.Repos(func(repoPath string) bool { return repoPath == id })
	if err != nil {
		return "", err
	}
	if len(repos) == 0 {
		return "", errRepoNotExist
	}
	return repos[0], nil
}

func (s *flatFileMultiRepoStore) Repos(f RepoFilter) ([]string, error) {
	if f == nil {
		f = allRepos
	}

	var repos []string
	w := fs.WalkFS(".", rwvfs.Walkable(s.fs))
	for w.Step() {
		if err := w.Err(); err != nil {
			return nil, err
		}
		fi := w.Stat()
		if fi.Mode().IsDir() {
			if fi.Name() == SrclibStoreDir {
				w.SkipDir()
				repo := path.Dir(w.Path())
				if f(repo) {
					repos = append(repos, repo)
				}
				continue
			}
			if strings.HasPrefix(fi.Name(), ".") {
				w.SkipDir()
				continue
			}
		}
	}
	return repos, nil
}

func (s *flatFileMultiRepoStore) getRepoStores() (map[string]RepoStore, error) {
	repos, err := s.Repos(nil)
	if err != nil {
		return nil, err
	}

	rss := make(map[string]RepoStore, len(repos))
	for _, repoPath := range repos {
		rss[repoPath] = NewFlatFileRepoStore(rwvfs.Sub(s.fs, path.Join(repoPath, SrclibStoreDir)), &FlatFileConfig{Codec: s.codec})
	}
	return rss, nil
}

func (s *flatFileMultiRepoStore) Import(repo, commitID string, unit *unit.SourceUnit, data graph.Output) error {
	repoPath := path.Join(repo, SrclibStoreDir)
	rs := NewFlatFileRepoStore(rwvfs.Sub(s.fs, repoPath), &FlatFileConfig{Codec: s.codec})
	return rs.Import(commitID, unit, data)
}

func (s *flatFileMultiRepoStore) String() string { return "flatFileMultiRepoStore" }

// A flatFileRepoStore is a RepoStore that stores data in flat files.
type flatFileRepoStore struct {
	fs rwvfs.FileSystem
	treeStores

	codec Codec
}

// NewFlatFileRepoStore creates a new repository store (that can be
// imported into) that is backed by files on a filesystem.
func NewFlatFileRepoStore(fs rwvfs.FileSystem, conf *FlatFileConfig) RepoStoreImporter {
	if conf == nil {
		conf = &FlatFileConfig{}
	}
	if conf.Codec == nil {
		conf.Codec = JSONCodec{}
	}

	setCreateParentDirs(fs)
	rs := &flatFileRepoStore{fs: fs, codec: conf.Codec}
	rs.treeStores = treeStores{treeStores: rs.getTreeStores}
	return rs
}

func (s *flatFileRepoStore) Version(key VersionKey) (*Version, error) {
	versions, err := s.Versions(versionCommitIDFilter(key.CommitID))
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
	ts := newFlatFileTreeStore(rwvfs.Sub(s.fs, commitID), &FlatFileConfig{Codec: s.codec})
	return ts.Import(unit, data)
}

func (s *flatFileRepoStore) getTreeStores() (map[string]TreeStore, error) {
	versionDirs, err := s.versionDirs()
	if err != nil {
		return nil, err
	}

	tss := make(map[string]TreeStore, len(versionDirs))
	for _, dir := range versionDirs {
		commitID := path.Base(dir)
		tss[commitID] = newFlatFileTreeStore(rwvfs.Sub(s.fs, commitID), &FlatFileConfig{Codec: s.codec})
	}
	return tss, nil
}

func (s *flatFileRepoStore) String() string { return "flatFileRepoStore" }

// A flatFileTreeStore is a TreeStore that stores data in flat files
// in a filesystem.
type flatFileTreeStore struct {
	fs rwvfs.FileSystem
	unitStores

	codec Codec
}

func newFlatFileTreeStore(fs rwvfs.FileSystem, conf *FlatFileConfig) *flatFileTreeStore {
	if conf == nil {
		conf = &FlatFileConfig{}
	}
	if conf.Codec == nil {
		conf.Codec = JSONCodec{}
	}

	ts := &flatFileTreeStore{fs: fs, codec: conf.Codec}
	ts.unitStores = unitStores{unitStores: ts.getUnitStores}
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
	return &unit, s.codec.Decode(f, &unit)
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
	return path.Join(key.Unit, key.UnitType+unitFileSuffix)
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
	if err := s.codec.Encode(f, unit); err != nil {
		return err
	}

	dir := strings.TrimSuffix(s.unitFilename(unit.Key()), unitFileSuffix)
	if err := rwvfs.MkdirAll(s.fs, dir); err != nil {
		return err
	}
	us := &flatFileUnitStore{fs: rwvfs.Sub(s.fs, dir), codec: s.codec}
	return us.Import(data)
}

func (s *flatFileTreeStore) getUnitStores() (map[unit.Key]UnitStore, error) {
	unitFiles, err := s.unitFilenames()
	if err != nil {
		return nil, err
	}

	uss := make(map[unit.Key]UnitStore, len(unitFiles))
	for _, unitFile := range unitFiles {
		dir := strings.TrimSuffix(unitFile, unitFileSuffix)
		unitKey := unit.Key{UnitType: path.Base(dir), Unit: path.Dir(dir)}
		uss[unitKey] = &flatFileUnitStore{fs: rwvfs.Sub(s.fs, dir), codec: s.codec}
	}
	return uss, nil
}

func (s *flatFileTreeStore) String() string { return "flatFileTreeStore" }

// A flatFileUnitStore is a UnitStore that stores data in flat files
// in a filesystem.
type flatFileUnitStore struct {
	fs rwvfs.FileSystem

	codec Codec
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
	cleanForUnitStoreImport(&data)
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
	return s.codec.Encode(f, data)
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
	return o, s.codec.Decode(f, o)
}

func (s *flatFileUnitStore) String() string { return "flatFileUnitStore" }

func setCreateParentDirs(fs rwvfs.FileSystem) {
	type createParents interface {
		CreateParentDirs(bool)
	}
	if fs, ok := fs.(createParents); ok {
		fs.CreateParentDirs(true)
	}
}
