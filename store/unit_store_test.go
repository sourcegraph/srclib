package store

import (
	"fmt"
	"reflect"
	"testing"

	"sourcegraph.com/sourcegraph/srclib/graph"
)

type unitStoreImporter interface {
	UnitStore
	UnitImporter
}

type labeledUnitStoreImporter struct {
	unitStoreImporter
	label string
}

func (s *labeledUnitStoreImporter) String() string {
	return fmt.Sprintf("%s: %s", s.unitStoreImporter, s.label)
}

func testUnitStore(t *testing.T, newFn func() unitStoreImporter) {
	testUnitStore_uninitialized(t, &labeledUnitStoreImporter{newFn(), "uninitialized"})
	testUnitStore_Import_empty(t, &labeledUnitStoreImporter{newFn(), "import empty"})
	testUnitStore_Import(t, &labeledUnitStoreImporter{newFn(), "import"})
	testUnitStore_Def(t, &labeledUnitStoreImporter{newFn(), "def"})
	testUnitStore_Defs(t, &labeledUnitStoreImporter{newFn(), "defs"})
	testUnitStore_Refs(t, &labeledUnitStoreImporter{newFn(), "refs"})
}

func testUnitStore_uninitialized(t *testing.T, us UnitStore) {
	def, err := us.Def(graph.DefKey{})
	if err == nil {
		t.Errorf("%s: Def: got nil err", us)
	}
	if def != nil {
		t.Errorf("%s: Def: got def %v, want nil", us, def)
	}

	defs, err := us.Defs(nil)
	if err == nil {
		t.Errorf("%s: Defs(nil): got nil err", us)
	}
	if len(defs) != 0 {
		t.Errorf("%s: Defs(nil): got defs %v, want empty", us, defs)
	}

	refs, err := us.Refs(nil)
	if err == nil {
		t.Errorf("%s: Refs(nil): got nil err", us)
	}
	if len(refs) != 0 {
		t.Errorf("%s: Refs(nil): got refs %v, want empty", us, refs)
	}
}

func testUnitStore_empty(t *testing.T, us UnitStore) {
	def, err := us.Def(graph.DefKey{})
	if !IsNotExist(err) {
		t.Errorf("%s: Def: got err %v, want IsNotExist-satisfying err", us, err)
	}
	if def != nil {
		t.Errorf("%s: Def: got def %v, want nil", us, def)
	}

	defs, err := us.Defs(nil)
	if err != nil {
		t.Errorf("%s: Defs(nil): %s", us, err)
	}
	if len(defs) != 0 {
		t.Errorf("%s: Defs(nil): got defs %v, want empty", us, defs)
	}

	refs, err := us.Refs(nil)
	if err != nil {
		t.Errorf("%s: Refs(nil): %s", us, err)
	}
	if len(refs) != 0 {
		t.Errorf("%s: Refs(nil): got refs %v, want empty", us, refs)
	}
}

func testUnitStore_Import_empty(t *testing.T, us unitStoreImporter) {
	data := graph.Output{
		Defs: []*graph.Def{},
		Refs: []*graph.Ref{},
	}
	if err := us.Import(data); err != nil {
		t.Errorf("%s: Import(empty data): %s", us, err)
	}
	testUnitStore_empty(t, us)
}

func testUnitStore_Import(t *testing.T, us unitStoreImporter) {
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
	if err := us.Import(data); err != nil {
		t.Errorf("%s: Import(data): %s", us, err)
	}
}

func testUnitStore_Def(t *testing.T, us unitStoreImporter) {
	data := graph.Output{
		Defs: []*graph.Def{
			{
				DefKey: graph.DefKey{Path: "p"},
				Name:   "n",
			},
		},
	}
	if err := us.Import(data); err != nil {
		t.Errorf("%s: Import(data): %s", us, err)
	}

	def, err := us.Def(graph.DefKey{Path: "p"})
	if err != nil {
		t.Errorf("%s: Def: %s", us, err)
	}
	if want := data.Defs[0]; !reflect.DeepEqual(def, want) {
		t.Errorf("%s: Def: got def %v, want %v", us, def, want)
	}

	def2, err := us.Def(graph.DefKey{Path: "p2"})
	if !IsNotExist(err) {
		t.Errorf("%s: Def: got err %v, want IsNotExist-satisfying err", us, err)
	}
	if def2 != nil {
		t.Errorf("%s: Def: got def %v, want nil", us, def2)
	}
}

func testUnitStore_Defs(t *testing.T, us unitStoreImporter) {
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
	if err := us.Import(data); err != nil {
		t.Errorf("%s: Import(data): %s", us, err)
	}

	defs, err := us.Defs(nil)
	if err != nil {
		t.Errorf("%s: Defs(nil): %s", us, err)
	}
	if want := data.Defs; !reflect.DeepEqual(defs, want) {
		t.Errorf("%s: Defs(nil): got defs %v, want %v", us, defs, want)
	}
}

func testUnitStore_Refs(t *testing.T, us unitStoreImporter) {
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
	if err := us.Import(data); err != nil {
		t.Errorf("%s: Import(data): %s", us, err)
	}

	refs, err := us.Refs(nil)
	if err != nil {
		t.Errorf("%s: Refs(nil): %s", us, err)
	}
	if want := data.Refs; !reflect.DeepEqual(refs, want) {
		t.Errorf("%s: Refs(nil): got refs %v, want %v", us, refs, want)
	}
}
