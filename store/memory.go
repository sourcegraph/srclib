package store

import (
	"errors"

	"sourcegraph.com/sourcegraph/srclib/graph"
	"sourcegraph.com/sourcegraph/srclib/unit"
)

// A memoryMultiRepoStore is a MultiRepoStore that stores data in
// memory
type memoryMultiRepoStore struct {
	repos map[string]*memoryRepoStore

	repoStores
}

func newMemoryMultiRepoStore() *memoryMultiRepoStore {
	mrs := &memoryMultiRepoStore{}
	mrs.repoStores = repoStores{repoStores: mrs.getRepoStores}
	return mrs
}

var errMultiRepoStoreNoInit = errors.New("multi-repo store not yet initialized")

func (s *memoryMultiRepoStore) Repo(id string) (string, error) {
	if s.repos == nil {
		return "", errMultiRepoStoreNoInit
	}

	_, present := s.repos[id]
	if !present {
		return "", errRepoNotExist
	}
	return id, nil
}

func (s *memoryMultiRepoStore) Repos(f RepoFilter) ([]string, error) {
	if f == nil {
		f = allRepos
	}

	if s.repos == nil {
		return nil, errMultiRepoStoreNoInit
	}

	var repos []string
	for repo := range s.repos {
		if f(repo) {
			repos = append(repos, repo)
		}
	}
	return repos, nil
}

func (s *memoryMultiRepoStore) getRepoStores() (map[string]RepoStore, error) {
	if s.repos == nil {
		return nil, errMultiRepoStoreNoInit
	}

	rss := make(map[string]RepoStore, len(s.repos))
	for id, rs := range s.repos {
		rss[id] = rs
	}
	return rss, nil
}

func (s *memoryMultiRepoStore) Import(repo, commitID string, unit *unit.SourceUnit, data graph.Output) error {
	if s.repos == nil {
		s.repos = map[string]*memoryRepoStore{}
	}
	if _, present := s.repos[repo]; !present {
		s.repos[repo] = newMemoryRepoStore()
	}
	return s.repos[repo].Import(commitID, unit, data)
}

func (s *memoryMultiRepoStore) String() string { return "memoryMultiRepoStore" }

// A memoryRepoStore is a RepoStore that stores data in memory.
type memoryRepoStore struct {
	versions []*Version
	trees    map[string]*memoryTreeStore
	treeStores
}

func newMemoryRepoStore() *memoryRepoStore {
	rs := &memoryRepoStore{}
	rs.treeStores = treeStores{treeStores: rs.getTreeStores}
	return rs
}

var errRepoNoInit = errors.New("repo not yet initialized")

func (s *memoryRepoStore) Version(key VersionKey) (*Version, error) {
	if s.versions == nil {
		return nil, errRepoNoInit
	}

	for _, version := range s.versions {
		if version.CommitID == key.CommitID {
			return version, nil
		}
	}
	return nil, errVersionNotExist
}

func (s *memoryRepoStore) Versions(f VersionFilter) ([]*Version, error) {
	if f == nil {
		f = allVersions
	}

	if s.versions == nil {
		return nil, errRepoNoInit
	}

	var versions []*Version
	for _, version := range s.versions {
		if f(version) {
			versions = append(versions, version)
		}

	}
	return versions, nil
}

func (s *memoryRepoStore) Import(commitID string, unit *unit.SourceUnit, data graph.Output) error {
	s.versions = append(s.versions, &Version{CommitID: commitID})
	if s.trees == nil {
		s.trees = map[string]*memoryTreeStore{}
	}
	if _, present := s.trees[commitID]; !present {
		s.trees[commitID] = newMemoryTreeStore()
	}
	return s.trees[commitID].Import(unit, data)
}

func (s *memoryRepoStore) getTreeStores() (map[string]TreeStore, error) {
	if s.trees == nil {
		return nil, errRepoNoInit
	}

	tss := make(map[string]TreeStore, len(s.trees))
	for commitID, ts := range s.trees {
		tss[commitID] = ts
	}
	return tss, nil
}

func (s *memoryRepoStore) String() string { return "memoryRepoStore" }

// A memoryTreeStore is a TreeStore that stores data in memory.
type memoryTreeStore struct {
	units []*unit.SourceUnit
	data  map[unit.Key]*graph.Output
	unitStores
}

func newMemoryTreeStore() *memoryTreeStore {
	ts := &memoryTreeStore{}
	ts.unitStores = unitStores{unitStores: ts.getUnitStores}
	return ts
}

var errTreeNoInit = errors.New("tree not yet initialized")

func (s *memoryTreeStore) Unit(key unit.Key) (*unit.SourceUnit, error) {
	if s.units == nil {
		return nil, errTreeNoInit
	}

	for _, unit := range s.units {
		if unit.Type == key.UnitType && unit.Name == key.Unit {
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
		s.data = map[unit.Key]*graph.Output{}
	}
	if u == nil {
		return nil
	}

	s.units = append(s.units, u)
	unitKey := unit.Key{UnitType: u.Type, Unit: u.Name}
	s.data[unitKey] = &data
	return nil
}

func (s *memoryTreeStore) getUnitStores() (map[unit.Key]UnitStore, error) {
	if s.data == nil {
		return nil, errTreeNoInit
	}

	uss := make(map[unit.Key]UnitStore, len(s.data))
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
	cleanForUnitStoreImport(&data)
	s.data = &data
	return nil
}

func (s *memoryUnitStore) String() string { return "memoryUnitStore" }
