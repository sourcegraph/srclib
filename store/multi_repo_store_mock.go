package store

type MockMultiRepoStore struct {
	Repo_  func(string) (string, error)
	Repos_ func(...RepoFilter) ([]string, error)
	MockRepoStore
}

func (m MockMultiRepoStore) Repo(uri string) (string, error) {
	return m.Repo_(uri)
}

func (m MockMultiRepoStore) Repos(f ...RepoFilter) ([]string, error) {
	return m.Repos_(f...)
}

var _ MultiRepoStore = MockMultiRepoStore{}
