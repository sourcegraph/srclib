package store

type MockRepoStore struct {
	Version_  func(VersionKey) (*Version, error)
	Versions_ func(...VersionFilter) ([]*Version, error)
	MockTreeStore
}

func (m MockRepoStore) Version(key VersionKey) (*Version, error) {
	return m.Version_(key)
}

func (m MockRepoStore) Versions(f ...VersionFilter) ([]*Version, error) {
	return m.Versions_(f...)
}

var _ RepoStore = MockRepoStore{}
