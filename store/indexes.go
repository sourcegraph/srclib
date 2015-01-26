package store

import "sourcegraph.com/sourcegraph/srclib/unit"

// IndexStatus describes an index and its status (whether it exists,
// etc.).
type IndexStatus struct {
	// Repo is the ID of the repository this index pertains to. If it
	// pertains to all repositories in a MultiRepoStore, or if it
	// pertains to the current (and only) repository in a RepoStore or
	// lower-level store, the Repo field is empty.
	Repo string

	// CommitID is the commit ID of the version this index pertains to. If it
	// pertains to all commits in a RepoStore, or if it
	// pertains to the current (and only) commit in a TreeStore or
	// lower-level store, the CommitID field is empty.
	CommitID string

	// Unit is the commit ID of the version this index pertains to. If
	// it pertains to all units in a TreeStore, or if it pertains to
	// the current (and only) source unit in a UnitStore, the Unit
	// field is empty.
	Unit unit.ID2

	// Stale is a boolean value indicating whether the index needs to
	// be (re)built.
	Stale bool

	// Path is the file path or URL to the index file (or directory,
	// if the index spans multiple files).
	Path string

	// Size is the length in bytes of the index if it is a regular
	// file.
	Size int64
}

// Indexes returns a list of indexes and their statuses for s and its
// lower-level stores.
func Indexes(s interface{}) ([]IndexStatus, error) {
	var xs []IndexStatus
	switch s := s.(type) {

	case *indexedTreeStore:

	case repoStoreOpener:
		rss, err := s.openAllRepoStores()
		if err != nil {
			return nil, err
		}
		for repo, rs := range rss {
			rxs, err := Indexes(rs)
			if err != nil {
				return nil, err
			}
			for _, x := range rxs {
				x.Repo = repo
				xs = append(xs, x)
			}
		}

	case treeStoreOpener:
		tss, err := s.openAllTreeStores()
		if err != nil {
			return nil, err
		}
		for tree, ts := range tss {
			rxs, err := Indexes(ts)
			if err != nil {
				return nil, err
			}
			for _, x := range rxs {
				x.CommitID = tree
				xs = append(xs, x)
			}
		}

	case unitStoreOpener:
		uss, err := s.openAllUnitStores()
		if err != nil {
			return nil, err
		}
		for unit, us := range uss {
			rxs, err := Indexes(us)
			if err != nil {
				return nil, err
			}
			for _, x := range rxs {
				x.Unit = unit
				xs = append(xs, x)
			}
		}

	}
	return xs, nil
}
