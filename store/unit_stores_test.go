package store

import (
	"reflect"
	"testing"

	"sourcegraph.com/sourcegraph/srclib/graph"
	"sourcegraph.com/sourcegraph/srclib/unit"
)

// mockNeverCalledUnitStore calls t.Error if any of its methods are
// called.
func mockNeverCalledUnitStore(t *testing.T) MockUnitStore {
	return MockUnitStore{
		Def_: func(key graph.DefKey) (*graph.Def, error) {
			t.Fatalf("(UnitStore).Def called, but wanted it not to be called (arg key was %+v)", key)
			return nil, nil
		},
		Defs_: func(f ...DefFilter) ([]*graph.Def, error) {
			t.Fatalf("(UnitStore).Defs called, but wanted it not to be called (arg f was %v)", f)
			return nil, nil
		},
		Refs_: func(f ...RefFilter) ([]*graph.Ref, error) {
			t.Fatalf("(UnitStore).Refs called, but wanted it not to be called (arg f was %v)", f)
			return nil, nil
		},
	}
}

type emptyUnitStore struct{}

func (m emptyUnitStore) Def(key graph.DefKey) (*graph.Def, error) {
	return nil, errDefNotExist
}

func (m emptyUnitStore) Defs(f ...DefFilter) ([]*graph.Def, error) {
	return []*graph.Def{}, nil
}

func (m emptyUnitStore) Refs(f ...RefFilter) ([]*graph.Ref, error) {
	return []*graph.Ref{}, nil
}

type mapUnitStoreOpener map[unitID]UnitStore

func (m mapUnitStoreOpener) openUnitStore(u unitID) (UnitStore, error) {
	if us, present := m[u]; present {
		return us, nil
	}
	return nil, errUnitNoInit
}
func (m mapUnitStoreOpener) openAllUnitStores() (map[unitID]UnitStore, error) { return m, nil }

type recordingUnitStoreOpener struct {
	opened    map[unitID]int // how many times openUnitStore was called for each unit
	openedAll int            // how many times openAllUnitStores was called
	unitStoreOpener
}

func (m *recordingUnitStoreOpener) openUnitStore(u unitID) (UnitStore, error) {
	if m.opened == nil {
		m.opened = map[unitID]int{}
	}
	m.opened[u]++
	return m.unitStoreOpener.openUnitStore(u)
}
func (m *recordingUnitStoreOpener) openAllUnitStores() (map[unitID]UnitStore, error) {
	m.openedAll++
	return m.unitStoreOpener.openAllUnitStores()
}
func (m *recordingUnitStoreOpener) reset() { m.opened = map[unitID]int{}; m.openedAll = 0 }

func TestUnitStores_filterByUnit(t *testing.T) {
	// Test that filters by source unit cause unit stores for other
	// source units to not be called.

	o := &recordingUnitStoreOpener{unitStoreOpener: mapUnitStoreOpener{
		unitID{"t", "u"}:  emptyUnitStore{},
		unitID{"t", "u2"}: mockNeverCalledUnitStore(t),
		unitID{"t2", "u"}: mockNeverCalledUnitStore(t),
	}}
	uss := unitStores{opener: o}

	if _, err := uss.Def(graph.DefKey{UnitType: "t", Unit: "u", Path: "p"}); !IsNotExist(err) {
		t.Errorf("got err %v, want IsNotExist-satisfying", err)
	}
	if want := map[unitID]int{unitID{"t", "u"}: 1}; !reflect.DeepEqual(o.opened, want) {
		t.Errorf("got opened %v, want %v", o.opened, want)
	}
	o.reset()

	if defs, err := uss.Defs(ByUnit("t", "u")); err != nil {
		t.Error(err)
	} else if len(defs) > 0 {
		t.Errorf("got defs %v, want none", defs)
	}

	if refs, err := uss.Refs(ByUnit("t", "u")); err != nil {
		t.Error(err)
	} else if len(refs) > 0 {
		t.Errorf("got refs %v, want none", refs)
	}
}

func TestScopeUnits(t *testing.T) {
	tests := []struct {
		filters []interface{}
		want    []unitID
	}{
		{
			filters: nil,
			want:    nil,
		},
		{
			filters: []interface{}{ByUnit("t", "u")},
			want:    []unitID{{"t", "u"}},
		},
		{
			filters: []interface{}{nil, ByUnit("t", "u")},
			want:    []unitID{{"t", "u"}},
		},
		{
			filters: []interface{}{ByUnit("t", "u"), nil},
			want:    []unitID{{"t", "u"}},
		},
		{
			filters: []interface{}{nil, ByUnit("t", "u"), nil},
			want:    []unitID{{"t", "u"}},
		},
		{
			filters: []interface{}{ByUnit("t", "u"), ByUnit("t", "u")},
			want:    []unitID{{"t", "u"}},
		},
		{
			filters: []interface{}{ByUnit("t1", "u1"), ByUnit("t2", "u2")},
			want:    []unitID{},
		},
		{
			filters: []interface{}{ByUnit("t1", "u1"), ByUnit("t2", "u2"), ByUnit("t1", "u1")},
			want:    []unitID{},
		},
		{
			filters: []interface{}{ByUnit("t", "u1"), ByUnit("t", "u2")},
			want:    []unitID{},
		},
		{
			filters: []interface{}{ByUnit("t1", "u"), ByUnit("t2", "u")},
			want:    []unitID{},
		},
		{
			filters: []interface{}{ByUnitKey(unit.Key{Repo: "r", CommitID: "c", UnitType: "t", Unit: "u"})},
			want:    []unitID{{"t", "u"}},
		},
		{
			filters: []interface{}{
				ByUnitKey(unit.Key{Repo: "r", CommitID: "c", UnitType: "t", Unit: "u"}),
				ByUnitKey(unit.Key{Repo: "r2", CommitID: "c2", UnitType: "t", Unit: "u"}),
			},
			want: []unitID{{"t", "u"}},
		},
		{
			filters: []interface{}{
				ByUnitKey(unit.Key{Repo: "r", CommitID: "c", UnitType: "t", Unit: "u"}),
				ByUnitKey(unit.Key{Repo: "r", CommitID: "c", UnitType: "t2", Unit: "u2"}),
			},
			want: []unitID{},
		},
		{
			filters: []interface{}{ByDefKey(graph.DefKey{Repo: "r", CommitID: "c", UnitType: "t", Unit: "u", Path: "p"})},
			want:    []unitID{{"t", "u"}},
		},
		{
			filters: []interface{}{
				ByDefKey(graph.DefKey{Repo: "r", CommitID: "c", UnitType: "t", Unit: "u", Path: "p"}),
				ByDefKey(graph.DefKey{Repo: "r2", CommitID: "c2", UnitType: "t", Unit: "u", Path: "p2"}),
			},
			want: []unitID{{"t", "u"}},
		},
		{
			filters: []interface{}{
				ByDefKey(graph.DefKey{Repo: "r", CommitID: "c", UnitType: "t", Unit: "u", Path: "p"}),
				ByDefKey(graph.DefKey{Repo: "r", CommitID: "c", UnitType: "t2", Unit: "u2", Path: "p"}),
			},
			want: []unitID{},
		},
		{
			filters: []interface{}{UnitFilterFunc(func(*unit.SourceUnit) bool { return false })},
			want:    nil,
		},
		{
			filters: []interface{}{ByRepo("r")},
			want:    nil,
		},
	}
	for _, test := range tests {
		units, err := scopeUnits(test.filters)
		if err != nil {
			t.Errorf("%+v: %v", test.filters, err)
			continue
		}
		if !reflect.DeepEqual(units, test.want) {
			t.Errorf("%+v: got units %v, want %v", test.filters, units, test.want)
		}
	}
}
