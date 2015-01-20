package store

// scopeRepos returns a list of repos that are matched by the
// filters. If potentially all repos could match, or if enough repos
// could potentially match that it would probably be cheaper to
// iterate through all of them, then a nil slice is returned. If none
// match, an empty slice is returned.
//
// TODO(sqs): return an error if the filters are mutually exclusive?
func scopeRepos(filters ...storesFilter) ([]string, error) {
	repos := map[string]struct{}{}

	for _, f := range filters {
		switch f := f.(type) {
		case ByRepoFilter:
			r := f.ByRepo()
			if len(repos) == 0 {
				repos[r] = struct{}{}
			} else if _, dup := repos[r]; !dup {
				// Mutually exclusive commit IDs.
				return []string{}, nil
			}
		}
	}

	if len(repos) == 0 {
		// Scope includes potentially all units.
		return nil, nil
	}

	repos2 := make([]string, 0, len(repos))
	for repo := range repos {
		repos2 = append(repos2, repo)
	}
	return repos2, nil
}

// A repoStoreOpener opens the RepoStore for the specified repo.
type repoStoreOpener interface {
	openRepoStore(repo string) (RepoStore, error)
	openAllRepoStores() (map[string]RepoStore, error)
}

// openRepoStores is a helper func that calls o.openRepoStore for each
// repo returned by scopeRepoStores(filters...).
func openRepoStores(o repoStoreOpener, filters ...storesFilter) (map[string]RepoStore, error) {
	repos, err := scopeRepos(filters...)
	if err != nil {
		return nil, err
	}

	if repos == nil {
		return o.openAllRepoStores()
	}

	rss := make(map[string]RepoStore, len(repos))
	for _, repo := range repos {
		var err error
		rss[repo], err = o.openRepoStore(repo)
		if err != nil {
			return nil, err
		}
	}
	return rss, nil
}
