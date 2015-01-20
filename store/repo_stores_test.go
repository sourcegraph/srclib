package store

import (
	"reflect"
	"testing"

	"sourcegraph.com/sourcegraph/srclib/graph"
	"sourcegraph.com/sourcegraph/srclib/unit"
)

type mockRepoStore struct {
	Version_  func(VersionKey) (*Version, error)
	Versions_ func(...VersionFilter) ([]*Version, error)
	mockTreeStore
}

func (m mockRepoStore) Version(key VersionKey) (*Version, error) {
	return m.Version_(key)
}

func (m mockRepoStore) Versions(f ...VersionFilter) ([]*Version, error) {
	return m.Versions_(f...)
}

// mockNeverCalledRepoStore calls t.Error if any of its methods are
// called.
func mockNeverCalledRepoStore(t *testing.T) RepoStore {
	return mockRepoStore{
		Version_: func(key VersionKey) (*Version, error) {
			t.Fatalf("(RepoStore).Version called, but wanted it not to be called (arg key was %+v)", key)
			return nil, nil
		},
		Versions_: func(f ...VersionFilter) ([]*Version, error) {
			t.Fatalf("(RepoStore).Versions called, but wanted it not to be called (arg f was %v)", f)
			return nil, nil
		},
		mockTreeStore: mockNeverCalledTreeStore(t),
	}
}

type emptyRepoStore struct{ emptyTreeStore }

func (m emptyRepoStore) Version(key VersionKey) (*Version, error) {
	return nil, errVersionNotExist
}

func (m emptyRepoStore) Versions(f ...VersionFilter) ([]*Version, error) {
	return []*Version{}, nil
}

type mapRepoStoreOpener map[string]RepoStore

func (m mapRepoStoreOpener) openRepoStore(repo string) (RepoStore, error) {
	if rs, present := m[repo]; present {
		return rs, nil
	}
	return nil, errRepoNoInit
}
func (m mapRepoStoreOpener) openAllRepoStores() (map[string]RepoStore, error) { return m, nil }

type recordingRepoStoreOpener struct {
	opened    map[string]int // how many times openRepoStore was called for each repo
	openedAll int            // how many times openAllRepoStores was called
	repoStoreOpener
}

func (m *recordingRepoStoreOpener) openRepoStore(repo string) (RepoStore, error) {
	if m.opened == nil {
		m.opened = map[string]int{}
	}
	m.opened[repo]++
	return m.repoStoreOpener.openRepoStore(repo)
}
func (m *recordingRepoStoreOpener) openAllRepoStores() (map[string]RepoStore, error) {
	m.openedAll++
	return m.repoStoreOpener.openAllRepoStores()
}
func (m *recordingRepoStoreOpener) reset() { m.opened = map[string]int{}; m.openedAll = 0 }

func TestRepoStores_filterByRepo(t *testing.T) {
	// Test that filters by a specific repo cause repo stores for
	// other repos to not be called.

	o := &recordingRepoStoreOpener{repoStoreOpener: mapRepoStoreOpener{
		"r":  emptyRepoStore{},
		"r2": mockNeverCalledRepoStore(t),
	}}
	rss := repoStores{opener: o}

	if _, err := rss.Def(graph.DefKey{Repo: "r", CommitID: "c", UnitType: "t", Unit: "u", Path: "p"}); !IsNotExist(err) {
		t.Errorf("got err %v, want IsNotExist-satisfying", err)
	}
	if want := map[string]int{"r": 1}; !reflect.DeepEqual(o.opened, want) {
		t.Errorf("got opened %v, want %v", o.opened, want)
	}
	o.reset()

	if defs, err := rss.Defs(ByRepo("r")); err != nil {
		t.Error(err)
	} else if len(defs) > 0 {
		t.Errorf("got defs %v, want none", defs)
	}

	if refs, err := rss.Refs(ByRepo("r")); err != nil {
		t.Error(err)
	} else if len(refs) > 0 {
		t.Errorf("got refs %v, want none", refs)
	}

	if _, err := rss.Unit(unit.Key{Repo: "r", CommitID: "c", UnitType: "t", Unit: "u"}); !IsNotExist(err) {
		t.Errorf("got err %v, want IsNotExist-satisfying", err)
	}

	if units, err := rss.Units(ByRepo("r")); err != nil {
		t.Error(err)
	} else if len(units) > 0 {
		t.Errorf("got units %v, want none", units)
	}

	if _, err := rss.Version(VersionKey{Repo: "r", CommitID: "c"}); !IsNotExist(err) {
		t.Errorf("got err %v, want IsNotExist-satisfying", err)
	}

	if versions, err := rss.Versions(ByRepo("r")); err != nil {
		t.Error(err)
	} else if len(versions) > 0 {
		t.Errorf("got versions %v, want none", versions)
	}
}

func TestScopeRepos(t *testing.T) {
	tests := []struct {
		filters []interface{}
		want    []string
	}{
		{
			filters: nil,
			want:    nil,
		},
		{
			filters: []interface{}{ByRepo("r")},
			want:    []string{"r"},
		},
		{
			filters: []interface{}{nil, ByRepo("r")},
			want:    []string{"r"},
		},
		{
			filters: []interface{}{ByRepo("r"), nil},
			want:    []string{"r"},
		},
		{
			filters: []interface{}{nil, ByRepo("r"), nil},
			want:    []string{"r"},
		},
		{
			filters: []interface{}{ByRepo("r"), ByRepo("r")},
			want:    []string{"r"},
		},
		{
			filters: []interface{}{ByRepo("r1"), ByRepo("r2")},
			want:    []string{},
		},
		{
			filters: []interface{}{ByRepo("r1"), ByRepo("r2"), ByRepo("r1")},
			want:    []string{},
		},
		{
			filters: []interface{}{ByUnitKey(unit.Key{Repo: "r", CommitID: "c", UnitType: "t", Unit: "u"})},
			want:    []string{"r"},
		},
		{
			filters: []interface{}{
				ByUnitKey(unit.Key{Repo: "r", CommitID: "c", UnitType: "t", Unit: "u"}),
				ByUnitKey(unit.Key{Repo: "r", CommitID: "c2", UnitType: "t2", Unit: "u2"}),
			},
			want: []string{"r"},
		},
		{
			filters: []interface{}{
				ByUnitKey(unit.Key{Repo: "r", CommitID: "c", UnitType: "t", Unit: "u"}),
				ByUnitKey(unit.Key{Repo: "r2", CommitID: "c", UnitType: "t", Unit: "u"}),
			},
			want: []string{},
		},
		{
			filters: []interface{}{ByDefKey(graph.DefKey{Repo: "r", CommitID: "c", UnitType: "t", Unit: "u", Path: "p"})},
			want:    []string{"r"},
		},
		{
			filters: []interface{}{
				ByDefKey(graph.DefKey{Repo: "r", CommitID: "c", UnitType: "t", Unit: "u", Path: "p"}),
				ByDefKey(graph.DefKey{Repo: "r", CommitID: "c2", UnitType: "t2", Unit: "u2", Path: "p2"}),
			},
			want: []string{"r"},
		},
		{
			filters: []interface{}{
				ByDefKey(graph.DefKey{Repo: "r", CommitID: "c", UnitType: "t", Unit: "u", Path: "p"}),
				ByDefKey(graph.DefKey{Repo: "r2", CommitID: "c", UnitType: "t", Unit: "u", Path: "p"}),
			},
			want: []string{},
		},
		{
			filters: []interface{}{RepoFilterFunc(func(string) bool { return false })},
			want:    nil,
		},
		{
			filters: []interface{}{ByUnit("t", "u")},
			want:    nil,
		},
	}
	for _, test := range tests {
		repos, err := scopeRepos(test.filters)
		if err != nil {
			t.Errorf("%+v: %v", test.filters, err)
			continue
		}
		if !reflect.DeepEqual(repos, test.want) {
			t.Errorf("%+v: got repos %v, want %v", test.filters, repos, test.want)
		}
	}
}
