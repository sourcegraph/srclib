package store

import (
	"fmt"
	"reflect"
	"testing"

	"sourcegraph.com/sourcegraph/srclib/graph"
	"sourcegraph.com/sourcegraph/srclib/unit"
)

type treeStoreImporter interface {
	TreeStore
	TreeImporter
}

type labeledTreeStoreImporter struct {
	treeStoreImporter
	label string
}

func (s *labeledTreeStoreImporter) String() string {
	return fmt.Sprintf("%s: %s", s.treeStoreImporter, s.label)
}

func testTreeStore(t *testing.T, newFn func() treeStoreImporter) {
	testTreeStore_uninitialized(t, &labeledTreeStoreImporter{newFn(), "uninitialized"})
	testTreeStore_Import_empty(t, &labeledTreeStoreImporter{newFn(), "import empty"})
	testTreeStore_Import(t, &labeledTreeStoreImporter{newFn(), "import"})
	testTreeStore_Unit(t, &labeledTreeStoreImporter{newFn(), "unit"})
	testTreeStore_Units(t, &labeledTreeStoreImporter{newFn(), "unit"})
	testTreeStore_Def(t, &labeledTreeStoreImporter{newFn(), "def"})
	testTreeStore_Defs(t, &labeledTreeStoreImporter{newFn(), "defs"})
	testTreeStore_Refs(t, &labeledTreeStoreImporter{newFn(), "refs"})
}

func testTreeStore_uninitialized(t *testing.T, ts treeStoreImporter) {
	unit, err := ts.Unit("t", "u")
	if err == nil {
		t.Errorf("%s: Unit: got nil err", ts)
	}
	if unit != nil {
		t.Errorf("%s: Unit: got unit %v, want nil", ts, unit)
	}

	units, err := ts.Units(nil)
	if err == nil {
		t.Errorf("%s: Units(nil): got nil err", ts)
	}
	if len(units) != 0 {
		t.Errorf("%s: Units(nil): got units %v, want empty", ts, units)
	}

	testUnitStore_uninitialized(t, ts)
}

func testTreeStore_empty(t *testing.T, ts treeStoreImporter) {
	unit, err := ts.Unit("t", "u")
	if !IsNotExist(err) {
		t.Errorf("%s: Unit: got err %v, want IsNotExist-satisfying err", ts, err)
	}
	if unit != nil {
		t.Errorf("%s: Unit: got unit %v, want nil", ts, unit)
	}

	units, err := ts.Units(nil)
	if err != nil {
		t.Errorf("%s: Units(nil): %s", ts, err)
	}
	if len(units) != 0 {
		t.Errorf("%s: Units(nil): got units %v, want empty", ts, units)
	}

	testUnitStore_empty(t, ts)
}

func testTreeStore_Import_empty(t *testing.T, ts treeStoreImporter) {
	if err := ts.Import(nil, graph.Output{}); err != nil {
		t.Errorf("%s: Import(nil, empty): %s", ts, err)
	}
	testTreeStore_empty(t, ts)
}

func testTreeStore_Import(t *testing.T, ts treeStoreImporter) {
	unit := &unit.SourceUnit{Type: "t", Name: "u"}
	data := graph.Output{
		Defs: []*graph.Def{
			{
				DefKey: graph.DefKey{Path: "p"},
				Name:   "n",
			},
		},
		Refs: []*graph.Ref{
			{
				DefPath: "p",
				File:    "f",
				Start:   1,
				End:     2,
			},
		},
	}
	if err := ts.Import(unit, data); err != nil {
		t.Errorf("%s: Import(%v, data): %s", ts, unit, err)
	}
}

func testTreeStore_Unit(t *testing.T, ts treeStoreImporter) {
	want := &unit.SourceUnit{Type: "t", Name: "u"}
	if err := ts.Import(want, graph.Output{}); err != nil {
		t.Errorf("%s: Import(%v, empty data): %s", ts, want, err)
	}

	unit, err := ts.Unit("t", "u")
	if err != nil {
		t.Errorf("%s: Unit(t, u): %s", ts, err)
	}
	if !reflect.DeepEqual(unit, want) {
		t.Errorf("%s: Unit(t, u): got %v, want %v", ts, unit, want)
	}
}

func testTreeStore_Units(t *testing.T, ts treeStoreImporter) {
	want := []*unit.SourceUnit{
		{Type: "t1", Name: "u1"},
		{Type: "t2", Name: "u2"},
	}
	for _, unit := range want {
		if err := ts.Import(unit, graph.Output{}); err != nil {
			t.Errorf("%s: Import(%v, empty data): %s", ts, unit, err)
		}
	}

	units, err := ts.Units(nil)
	if err != nil {
		t.Errorf("%s: Units(nil): %s", ts, err)
	}
	if !reflect.DeepEqual(units, want) {
		t.Errorf("%s: Units(nil): got %v, want %v", ts, units, want)
	}
}

func testTreeStore_Def(t *testing.T, ts treeStoreImporter) {
	unit := &unit.SourceUnit{Type: "t", Name: "u"}
	data := graph.Output{
		Defs: []*graph.Def{
			{
				DefKey: graph.DefKey{Path: "p"},
				Name:   "n",
			},
		},
	}
	if err := ts.Import(unit, data); err != nil {
		t.Errorf("%s: Import(%v, data): %s", ts, unit, err)
	}

	def, err := ts.Def(graph.DefKey{Path: "p"})
	if !IsNotExist(err) {
		t.Errorf("%s: Def(no unit): got err %v, want IsNotExist-satisfying err", ts, err)
	}
	if def != nil {
		t.Errorf("%s: Def: got def %v, want nil", ts, def)
	}

	def, err = ts.Def(graph.DefKey{UnitType: "t", Unit: "u", Path: "p"})
	if err != nil {
		t.Errorf("%s: Def: %s", ts, err)
	}
	if want := data.Defs[0]; !reflect.DeepEqual(def, want) {
		t.Errorf("%s: Def: got def %v, want %v", ts, def, want)
	}

	def2, err := ts.Def(graph.DefKey{UnitType: "t", Unit: "u", Path: "p2"})
	if !IsNotExist(err) {
		t.Errorf("%s: Def: got err %v, want IsNotExist-satisfying err", ts, err)
	}
	if def2 != nil {
		t.Errorf("%s: Def: got def %v, want nil", ts, def2)
	}
}

func testTreeStore_Defs(t *testing.T, ts treeStoreImporter) {
	unit := &unit.SourceUnit{Type: "t", Name: "u"}
	data := graph.Output{
		Defs: []*graph.Def{
			{
				DefKey: graph.DefKey{Path: "p1"},
				Name:   "n1",
			},
			{
				DefKey: graph.DefKey{Path: "p2"},
				Name:   "n2",
			},
		},
	}
	if err := ts.Import(unit, data); err != nil {
		t.Errorf("%s: Import(%v, data): %s", ts, unit, err)
	}

	want := []*graph.Def{
		{
			DefKey: graph.DefKey{UnitType: "t", Unit: "u", Path: "p1"},
			Name:   "n1",
		},
		{
			DefKey: graph.DefKey{UnitType: "t", Unit: "u", Path: "p2"},
			Name:   "n2",
		},
	}

	defs, err := ts.Defs(nil)
	if err != nil {
		t.Errorf("%s: Defs(nil): %s", ts, err)
	}
	if !reflect.DeepEqual(defs, want) {
		t.Errorf("%s: Defs(nil): got defs %v, want %v", ts, defs, want)
	}
}

func testTreeStore_Refs(t *testing.T, ts treeStoreImporter) {
	unit := &unit.SourceUnit{Type: "t", Name: "u"}
	data := graph.Output{
		Refs: []*graph.Ref{
			{
				DefPath: "p1",
				File:    "f1",
				Start:   1,
				End:     2,
			},
			{
				DefPath: "p2",
				File:    "f2",
				Start:   2,
				End:     3,
			},
		},
	}
	if err := ts.Import(unit, data); err != nil {
		t.Errorf("%s: Import(%v, data): %s", ts, unit, err)
	}

	want := []*graph.Ref{
		{
			DefPath:  "p1",
			File:     "f1",
			Start:    1,
			End:      2,
			UnitType: "t", Unit: "u",
		},
		{
			DefPath:  "p2",
			File:     "f2",
			Start:    2,
			End:      3,
			UnitType: "t", Unit: "u",
		},
	}

	refs, err := ts.Refs(nil)
	if err != nil {
		t.Errorf("%s: Refs(nil): %s", ts, err)
	}
	if !reflect.DeepEqual(refs, want) {
		t.Errorf("%s: Refs(nil): got refs %v, want %v", ts, refs, want)
	}
}
