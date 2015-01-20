package store

// scopeTrees returns a list of commit IDs that are matched by the
// filters. If potentially all commits could match, or if enough
// commits could potentially match that it would probably be cheaper
// to iterate through all of them, then a nil slice is returned. If
// none match, an empty slice is returned.
//
// scopeTrees is used to select which TreeStores to query.
//
// TODO(sqs): return an error if the filters are mutually exclusive?
func scopeTrees(filters []interface{}) ([]string, error) {
	commitIDs := map[string]struct{}{}

	for _, f := range filters {
		switch f := f.(type) {
		case ByCommitIDFilter:
			c := f.ByCommitID()
			if len(commitIDs) == 0 {
				commitIDs[c] = struct{}{}
			} else if _, dup := commitIDs[c]; !dup {
				// Mutually exclusive commit IDs.
				return []string{}, nil
			}
		}
	}

	if len(commitIDs) == 0 {
		// Scope includes potentially all units.
		return nil, nil
	}

	ids := make([]string, 0, len(commitIDs))
	for commitID := range commitIDs {
		ids = append(ids, commitID)
	}
	return ids, nil
}

// A treeStoreOpener opens the TreeStore for the specified tree.
type treeStoreOpener interface {
	openTreeStore(commitID string) (TreeStore, error)
	openAllTreeStores() (map[string]TreeStore, error)
}

// openCommitstores is a helper func that calls o.openTreeStore for
// each tree returned by scopeTrees(filters...).
func openTreeStores(o treeStoreOpener, filters interface{}) (map[string]TreeStore, error) {
	commitIDs, err := scopeTrees(storeFilters(filters))
	if err != nil {
		return nil, err
	}

	if commitIDs == nil {
		return o.openAllTreeStores()
	}

	tss := make(map[string]TreeStore, len(commitIDs))
	for _, commitID := range commitIDs {
		var err error
		tss[commitID], err = o.openTreeStore(commitID)
		if err != nil {
			return nil, err
		}
	}
	return tss, nil
}
