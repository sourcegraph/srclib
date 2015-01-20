package store

import (
	"reflect"
	"testing"

	"sourcegraph.com/sourcegraph/srclib/graph"
	"sourcegraph.com/sourcegraph/srclib/unit"
)

type mockTreeStore struct {
	Unit_  func(unit.Key) (*unit.SourceUnit, error)
	Units_ func(UnitFilter) ([]*unit.SourceUnit, error)
	mockUnitStore
}

func (m mockTreeStore) Unit(key unit.Key) (*unit.SourceUnit, error) {
	return m.Unit_(key)
}

func (m mockTreeStore) Units(f UnitFilter) ([]*unit.SourceUnit, error) {
	return m.Units_(f)
}

// mockNeverCalledTreeStore calls t.Error if any of its methods are
// called.
func mockNeverCalledTreeStore(t *testing.T) mockTreeStore {
	return mockTreeStore{
		Unit_: func(key unit.Key) (*unit.SourceUnit, error) {
			t.Fatalf("(TreeStore).Unit called, but wanted it not to be called (arg key was %+v)", key)
			return nil, nil
		},
		Units_: func(f UnitFilter) ([]*unit.SourceUnit, error) {
			t.Fatalf("(TreeStore).Units called, but wanted it not to be called (arg f was %v)", f)
			return nil, nil
		},
		mockUnitStore: mockNeverCalledUnitStore(t),
	}
}

type emptyTreeStore struct{ emptyUnitStore }

func (m emptyTreeStore) Unit(key unit.Key) (*unit.SourceUnit, error) {
	return nil, errUnitNotExist
}

func (m emptyTreeStore) Units(f UnitFilter) ([]*unit.SourceUnit, error) {
	return []*unit.SourceUnit{}, nil
}

type mapTreeStoreOpener map[string]TreeStore

func (m mapTreeStoreOpener) openTreeStore(commitID string) (TreeStore, error) {
	if ts, present := m[commitID]; present {
		return ts, nil
	}
	return nil, errTreeNoInit
}
func (m mapTreeStoreOpener) openAllTreeStores() (map[string]TreeStore, error) { return m, nil }

type recordingTreeStoreOpener struct {
	opened    map[string]int // how many times openTreeStore was called for each tree
	openedAll int            // how many times openAllTreeStores was called
	treeStoreOpener
}

func (m *recordingTreeStoreOpener) openTreeStore(commitID string) (TreeStore, error) {
	if m.opened == nil {
		m.opened = map[string]int{}
	}
	m.opened[commitID]++
	return m.treeStoreOpener.openTreeStore(commitID)
}
func (m *recordingTreeStoreOpener) openAllTreeStores() (map[string]TreeStore, error) {
	m.openedAll++
	return m.treeStoreOpener.openAllTreeStores()
}
func (m *recordingTreeStoreOpener) reset() { m.opened = map[string]int{}; m.openedAll = 0 }

func TestTreeStores_filterByCommit(t *testing.T) {
	// Test that filters by a specific commit cause tree stores for
	// other commits to not be called.

	o := &recordingTreeStoreOpener{treeStoreOpener: mapTreeStoreOpener{
		"c":  emptyTreeStore{},
		"c2": mockNeverCalledTreeStore(t),
	}}
	tss := treeStores{opener: o}

	if _, err := tss.Def(graph.DefKey{CommitID: "c", UnitType: "t", Unit: "u", Path: "p"}); !IsNotExist(err) {
		t.Errorf("got err %v, want IsNotExist-satisfying", err)
	}
	if want := map[string]int{"c": 1}; !reflect.DeepEqual(o.opened, want) {
		t.Errorf("got opened %v, want %v", o.opened, want)
	}
	o.reset()

	if defs, err := tss.Defs(ByCommitID("c")); err != nil {
		t.Error(err)
	} else if len(defs) > 0 {
		t.Errorf("got defs %v, want none", defs)
	}

	if refs, err := tss.Refs(ByCommitID("c")); err != nil {
		t.Error(err)
	} else if len(refs) > 0 {
		t.Errorf("got refs %v, want none", refs)
	}

	if _, err := tss.Unit(unit.Key{CommitID: "c", UnitType: "t", Unit: "u"}); !IsNotExist(err) {
		t.Errorf("got err %v, want IsNotExist-satisfying", err)
	}

	if units, err := tss.Units(ByCommitID("c")); err != nil {
		t.Error(err)
	} else if len(units) > 0 {
		t.Errorf("got units %v, want none", units)
	}
}

func TestScopeTrees(t *testing.T) {
	tests := []struct {
		filters []storesFilter
		want    []string
	}{
		{
			filters: nil,
			want:    nil,
		},
		{
			filters: []storesFilter{ByCommitID("c")},
			want:    []string{"c"},
		},
		{
			filters: []storesFilter{nil, ByCommitID("c")},
			want:    []string{"c"},
		},
		{
			filters: []storesFilter{ByCommitID("c"), nil},
			want:    []string{"c"},
		},
		{
			filters: []storesFilter{nil, ByCommitID("c"), nil},
			want:    []string{"c"},
		},
		{
			filters: []storesFilter{ByCommitID("c"), ByCommitID("c")},
			want:    []string{"c"},
		},
		{
			filters: []storesFilter{ByCommitID("c1"), ByCommitID("c2")},
			want:    []string{},
		},
		{
			filters: []storesFilter{ByCommitID("c1"), ByCommitID("c2"), ByCommitID("c1")},
			want:    []string{},
		},
		{
			filters: []storesFilter{ByUnitKey(unit.Key{Repo: "r", CommitID: "c", UnitType: "t", Unit: "u"})},
			want:    []string{"c"},
		},
		{
			filters: []storesFilter{
				ByUnitKey(unit.Key{Repo: "r", CommitID: "c", UnitType: "t", Unit: "u"}),
				ByUnitKey(unit.Key{Repo: "r2", CommitID: "c", UnitType: "t2", Unit: "u2"}),
			},
			want: []string{"c"},
		},
		{
			filters: []storesFilter{
				ByUnitKey(unit.Key{Repo: "r", CommitID: "c", UnitType: "t", Unit: "u"}),
				ByUnitKey(unit.Key{Repo: "r", CommitID: "c2", UnitType: "t", Unit: "u"}),
			},
			want: []string{},
		},
		{
			filters: []storesFilter{ByDefKey(graph.DefKey{Repo: "r", CommitID: "c", UnitType: "t", Unit: "u", Path: "p"})},
			want:    []string{"c"},
		},
		{
			filters: []storesFilter{
				ByDefKey(graph.DefKey{Repo: "r", CommitID: "c", UnitType: "t", Unit: "u", Path: "p"}),
				ByDefKey(graph.DefKey{Repo: "r2", CommitID: "c", UnitType: "t2", Unit: "u2", Path: "p2"}),
			},
			want: []string{"c"},
		},
		{
			filters: []storesFilter{
				ByDefKey(graph.DefKey{Repo: "r", CommitID: "c", UnitType: "t", Unit: "u", Path: "p"}),
				ByDefKey(graph.DefKey{Repo: "r", CommitID: "c2", UnitType: "t", Unit: "u", Path: "p"}),
			},
			want: []string{},
		},
		{
			filters: []storesFilter{VersionFilterFunc(func(*Version) bool { return false })},
			want:    nil,
		},
		{
			filters: []storesFilter{ByUnit("t", "u")},
			want:    nil,
		},
	}
	for _, test := range tests {
		trees, err := scopeTrees(test.filters...)
		if err != nil {
			t.Errorf("%+v: %v", test.filters, err)
			continue
		}
		if !reflect.DeepEqual(trees, test.want) {
			t.Errorf("%+v: got trees %v, want %v", test.filters, trees, test.want)
		}
	}
}
