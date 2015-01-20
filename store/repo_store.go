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

// A VersionFilter is used to filter a list of versions to only those
// for which the func returns true.
type VersionFilter func(*Version) bool

// allVersions is a VersionFilter that selects all versions.
func allVersions(*Version) bool { return true }

// versionCommitIDFilter selects all versions whose CommitID equals
// the given commitID.
func versionCommitIDFilter(commitID string) VersionFilter {
	return func(version *Version) bool {
		return version.CommitID == commitID
	}
}

// A repoStores is a RepoStore whose methods call the
// corresponding method on each of the repo stores returned by the
// repoStores func.
type repoStores struct {
	repoStores func() (map[string]RepoStore, error)
}

var _ RepoStore = (*repoStores)(nil)

func (s repoStores) Version(key VersionKey) (*Version, error) {
	rss, err := s.repoStores()
	if err != nil {
		return nil, err
	}

	for repo, rs := range rss {
		if key.Repo != repo {
			continue
		}
		version, err := rs.Version(key)
		if err != nil {
			if IsNotExist(err) {
				continue
			}
			return nil, err
		}
		if version.Repo == "" {
			version.Repo = repo
		}
		return version, nil
	}
	return nil, errVersionNotExist
}

func (s repoStores) Versions(f VersionFilter) ([]*Version, error) {
	if f == nil {
		f = allVersions
	}

	rss, err := s.repoStores()
	if err != nil {
		return nil, err
	}

	var allVersions []*Version
	for repo, rs := range rss {
		versions, err := rs.Versions(func(version *Version) bool {
			if version.Repo == "" {
				version.Repo = repo
			}
			return f(version)
		})
		if err != nil {
			return nil, err
		}
		allVersions = append(allVersions, versions...)
	}
	return allVersions, nil
}

func (s repoStores) Unit(key unit.Key) (*unit.SourceUnit, error) {
	rss, err := s.repoStores()
	if err != nil {
		return nil, err
	}

	for repo, rs := range rss {
		if key.Repo != repo {
			continue
		}
		unit, err := rs.Unit(key)
		if err != nil {
			if IsNotExist(err) {
				continue
			}
			return nil, err
		}
		if unit.Repo == "" {
			unit.Repo = repo
		}
		return unit, nil
	}
	return nil, errUnitNotExist
}

func (s repoStores) Units(f UnitFilter) ([]*unit.SourceUnit, error) {
	if f == nil {
		f = allUnits
	}

	rss, err := s.repoStores()
	if err != nil {
		return nil, err
	}

	var allUnits []*unit.SourceUnit
	for repo, rs := range rss {
		units, err := rs.Units(func(unit *unit.SourceUnit) bool {
			if unit.Repo == "" {
				unit.Repo = repo
			}
			return f(unit)
		})
		if err != nil {
			return nil, err
		}
		allUnits = append(allUnits, units...)
	}
	return allUnits, nil
}

func (s repoStores) Def(key graph.DefKey) (*graph.Def, error) {
	rss, err := s.repoStores()
	if err != nil {
		return nil, err
	}

	for repo, rs := range rss {
		if key.Repo != repo {
			continue
		}
		key.Repo = ""
		def, err := rs.Def(key)
		if err != nil {
			if IsNotExist(err) {
				continue
			}
			return nil, err
		}
		if def.Repo == "" {
			def.Repo = repo
		}
		return def, nil
	}
	return nil, errDefNotExist
}

func (s repoStores) Defs(f DefFilter) ([]*graph.Def, error) {
	if f == nil {
		f = allDefs
	}

	rss, err := s.repoStores()
	if err != nil {
		return nil, err
	}

	var allDefs []*graph.Def
	for repo, rs := range rss {
		defs, err := rs.Defs(func(def *graph.Def) bool {
			if def.Repo == "" {
				def.Repo = repo
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

func (s repoStores) Refs(f RefFilter) ([]*graph.Ref, error) {
	if f == nil {
		f = allRefs
	}

	rss, err := s.repoStores()
	if err != nil {
		return nil, err
	}
	var allRefs []*graph.Ref
	for repo, rs := range rss {
		refs, err := rs.Refs(func(ref *graph.Ref) bool {
			if ref.Repo == "" {
				ref.Repo = repo
			}
			if ref.DefRepo == "" {
				ref.DefRepo = repo
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
