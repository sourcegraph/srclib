package store

import (
	"sourcegraph.com/sourcegraph/srclib/graph"
	"sourcegraph.com/sourcegraph/srclib/unit"
)

// A RepoStore stores and accesses srclib build data for a repository
// (consisting of any number of commits, each of which have any number
// of source units).
type RepoStore interface {
	// Version gets the tree for a single version (i.e., commit).
	Version(commitID string) (*Version, error)

	// Version gets the tree for a single version (i.e., commit).
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

// A Version represents a revision (i.e., commit) of a repository.
type Version struct {
	// CommitID is the commit ID of the VCS revision that this version
	// represents. If blank, then this version refers to the current
	// workspace.
	CommitID string
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
