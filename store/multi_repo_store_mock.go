package store

type MockMultiRepoStore struct {
	Repos_ func(...RepoFilter) ([]string, error)
	MockRepoStore
}

func (m MockMultiRepoStore) Repos(f ...RepoFilter) ([]string, error) {
	return m.Repos_(f...)
}

var _ MultiRepoStore = MockMultiRepoStore{}
