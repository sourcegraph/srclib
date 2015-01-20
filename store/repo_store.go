package store

import (
	"sourcegraph.com/sourcegraph/srclib/graph"
	"sourcegraph.com/sourcegraph/srclib/unit"
)

// A RepoStore stores and accesses srclib build data for a repository
// (consisting of any number of commits, each of which have any number
// of source units).
type RepoStore interface {
	// Version gets a single commit.
	Version(VersionKey) (*Version, error)

	// Versions returns all commits that match the VersionFilter.
	Versions(VersionFilter) ([]*Version, error)

	// TreeStore's methods call the corresponding methods on the
	// TreeStore of each version contained within this repository. The
	// combined results are returned (in undefined order).
	TreeStore
}

// A RepoImporter imports srclib build data for a source unit at a
// specific version into a RepoStore.
type RepoImporter interface {
	// Import imports srclib build data for a source unit at a
	// specific version into the store.
	Import(commitID string, unit *unit.SourceUnit, data graph.Output) error
}

// A RepoStoreImporter implements both RepoStore and RepoImporter.
type RepoStoreImporter interface {
	RepoStore
	RepoImporter
}

// A VersionKey is a unique identifier for a version across all
// repositories.
type VersionKey struct {
	// Repo is the URI of the commit's repository.
	Repo string

	// CommitID is the commit ID of the commit.
	CommitID string
}

// A Version represents a revision (i.e., commit) of a repository.
type Version struct {
	// Repo is the URI of the repository that contains this commit.
	Repo string

	// CommitID is the commit ID of the VCS revision that this version
	// represents. If blank, then this version refers to the current
	// workspace.
	CommitID string

	// TODO(sqs): add build metadata fields (build logs, timings, what
	// was actually built, incremental build tracking, diff/pack
	// compression helper info, etc.)
}

// IsCurrentWorkspace returns a boolean indicating whether this
// version represents the current workspace, as opposed to a specific
// VCS commit.
func (v Version) IsCurrentWorkspace() bool { return v.CommitID == "" }

// A repoStores is a RepoStore whose methods call the
// corresponding method on each of the repo stores returned by the
// repoStores func.
type repoStores struct {
	opener repoStoreOpener
}

var _ RepoStore = (*repoStores)(nil)

func (s repoStores) Version(key VersionKey) (*Version, error) {
	rss, err := openRepoStores(s.opener, ByRepoAndCommitID(key.Repo, key.CommitID))
	if err != nil {
		if isStoreNotExist(err) {
			return nil, errVersionNotExist
		}
		return nil, err
	}

	for repo, rs := range rss {
		if key.Repo != repo {
			continue
		}
		version, err := rs.Version(key)
		if err != nil {
			if IsNotExist(err) || isStoreNotExist(err) {
				continue
			}
			return nil, err
		}
		version.Repo = repo
		return version, nil
	}
	return nil, errVersionNotExist
}

func (s repoStores) Versions(f VersionFilter) ([]*Version, error) {
	rss, err := openRepoStores(s.opener, f)
	if err != nil {
		return nil, err
	}

	if f == nil {
		f = allVersions
	}

	var allVersions []*Version
	for repo, rs := range rss {
		versions, err := rs.Versions(f)
		if err != nil {
			return nil, err
		}
		for _, version := range versions {
			version.Repo = repo
		}
		allVersions = append(allVersions, versions...)
	}
	return allVersions, nil
}

func (s repoStores) Unit(key unit.Key) (*unit.SourceUnit, error) {
	rss, err := openRepoStores(s.opener, ByUnitKey(key))
	if err != nil {
		if isStoreNotExist(err) {
			return nil, errUnitNotExist
		}
		return nil, err
	}

	for repo, rs := range rss {
		if key.Repo != repo {
			continue
		}
		unit, err := rs.Unit(key)
		if err != nil {
			if IsNotExist(err) || isStoreNotExist(err) {
				continue
			}
			return nil, err
		}
		unit.Repo = repo
		return unit, nil
	}
	return nil, errUnitNotExist
}

func (s repoStores) Units(f UnitFilter) ([]*unit.SourceUnit, error) {
	rss, err := openRepoStores(s.opener, f)
	if err != nil {
		return nil, err
	}

	if f == nil {
		f = allUnits
	}

	var allUnits []*unit.SourceUnit
	for repo, rs := range rss {
		units, err := rs.Units(f)
		if err != nil {
			return nil, err
		}
		for _, unit := range units {
			unit.Repo = repo
		}
		allUnits = append(allUnits, units...)
	}
	return allUnits, nil
}

func (s repoStores) Def(key graph.DefKey) (*graph.Def, error) {
	if err := checkDefKeyValidForMultiRepoStore(key); err != nil {
		return nil, err
	}

	rss, err := openRepoStores(s.opener, ByDefKey(key))
	if err != nil {
		if isStoreNotExist(err) {
			return nil, errDefNotExist
		}
		return nil, err
	}

	for repo, rs := range rss {
		if key.Repo != repo {
			continue
		}
		def, err := rs.Def(key)
		if err != nil {
			if IsNotExist(err) || isStoreNotExist(err) {
				continue
			}
			return nil, err
		}
		def.Repo = repo
		return def, nil
	}
	return nil, errDefNotExist
}

func (s repoStores) Defs(f DefFilter) ([]*graph.Def, error) {
	rss, err := openRepoStores(s.opener, f)
	if err != nil {
		return nil, err
	}

	if f == nil {
		f = allDefs
	}

	var allDefs []*graph.Def
	for repo, rs := range rss {
		defs, err := rs.Defs(f)
		if err != nil {
			return nil, err
		}
		for _, def := range defs {
			def.Repo = repo
		}
		allDefs = append(allDefs, defs...)
	}
	return allDefs, nil
}

func (s repoStores) Refs(f RefFilter) ([]*graph.Ref, error) {
	rss, err := openRepoStores(s.opener, f)
	if err != nil {
		return nil, err
	}

	if f == nil {
		f = allRefs
	}

	var allRefs []*graph.Ref
	for repo, rs := range rss {
		refs, err := rs.Refs(f)
		if err != nil {
			return nil, err
		}
		for _, ref := range refs {
			ref.Repo = repo
			if ref.DefRepo == "" {
				ref.DefRepo = repo
			}
		}
		allRefs = append(allRefs, refs...)
	}
	return allRefs, nil
}

// checkDefKeyValidForRepoStore returns an *InvalidKeyError if the def
// key is underspecified for use in (RepoStore).Def.
func checkDefKeyValidForRepoStore(key graph.DefKey) error {
	if err := checkDefKeyValidForTreeStore(key); err != nil {
		return err
	}
	if key.CommitID == "" {
		return &InvalidKeyError{"empty DefKey.CommitID"}
	}
	return nil
}
