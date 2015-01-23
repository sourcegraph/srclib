package store

import (
	"fmt"
	"reflect"
	"testing"

	"sort"

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
	testUnitStore_Refs_ByFile(t, &labeledUnitStoreImporter{newFn(), "refs by file"})
}

func testUnitStore_uninitialized(t *testing.T, us UnitStore) {
	def, err := us.Def(graph.DefKey{Repo: "r", CommitID: "c", UnitType: "t", Unit: "u", Path: "p"})
	if err == nil {
		t.Errorf("%s: Def: got nil err", us)
	}
	if def != nil {
		t.Errorf("%s: Def: got def %v, want nil", us, def)
	}

	defs, err := us.Defs()
	if err == nil {
		t.Errorf("%s: Defs(): got nil err", us)
	}
	if len(defs) != 0 {
		t.Errorf("%s: Defs(): got defs %v, want empty", us, defs)
	}

	refs, err := us.Refs()
	if err == nil {
		t.Errorf("%s: Refs(): got nil err", us)
	}
	if len(refs) != 0 {
		t.Errorf("%s: Refs(): got refs %v, want empty", us, refs)
	}
}

func testUnitStore_empty(t *testing.T, us UnitStore) {
	def, err := us.Def(graph.DefKey{Repo: "r", CommitID: "c", UnitType: "t", Unit: "u", Path: "p"})
	if !IsNotExist(err) {
		t.Errorf("%s: Def: got err %v, want IsNotExist-satisfying err", us, err)
	}
	if def != nil {
		t.Errorf("%s: Def: got def %v, want nil", us, def)
	}

	defs, err := us.Defs()
	if err != nil {
		t.Errorf("%s: Defs(): %s", us, err)
	}
	if len(defs) != 0 {
		t.Errorf("%s: Defs(): got defs %v, want empty", us, defs)
	}

	refs, err := us.Refs()
	if err != nil {
		t.Errorf("%s: Refs(): %s", us, err)
	}
	if len(refs) != 0 {
		t.Errorf("%s: Refs(): got refs %v, want empty", us, refs)
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

	defs, err := us.Defs()
	if err != nil {
		t.Errorf("%s: Defs(): %s", us, err)
	}
	if want := data.Defs; !reflect.DeepEqual(defs, want) {
		t.Errorf("%s: Defs(): got defs %v, want %v", us, defs, want)
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

	refs, err := us.Refs()
	if err != nil {
		t.Errorf("%s: Refs(): %s", us, err)
	}
	if want := data.Refs; !reflect.DeepEqual(refs, want) {
		t.Errorf("%s: Refs(): got refs %v, want %v", us, refs, want)
	}
}

func testUnitStore_Refs_ByFile(t *testing.T, us unitStoreImporter) {
	refsByFile := map[string][]*graph.Ref{
		"f1": {
			{DefPath: "p1", Start: 0, End: 5},
		},
		"f2": {
			{DefPath: "p1", Start: 0, End: 5},
			{DefPath: "p2", Start: 5, End: 10},
		},
		"f3": {
			{DefPath: "p1", Start: 0, End: 5},
			{DefPath: "p2", Start: 5, End: 10},
			{DefPath: "p3", Start: 10, End: 15},
		},
	}
	var data graph.Output
	for file, refs := range refsByFile {
		for _, ref := range refs {
			ref.File = file
		}
		data.Refs = append(data.Refs, refs...)
	}

	if err := us.Import(data); err != nil {
		t.Errorf("%s: Import(data): %s", us, err)
	}

	for file, wantRefs := range refsByFile {
		c_refFileIndex_getByFile = 0
		refs, err := us.Refs(ByFile(file))
		if err != nil {
			t.Fatalf("%s: Refs(ByFile %s): %s", us, file, err)
		}
		sort.Sort(refsByFileStartEnd(refs))
		sort.Sort(refsByFileStartEnd(wantRefs))
		if want := wantRefs; !reflect.DeepEqual(refs, want) {
			t.Errorf("%s: Refs(ByFile %s): got refs %v, want %v", us, file, refs, want)
		}
		if isIndexedStore(us) {
			if want := 1; c_refFileIndex_getByFile != want {
				t.Errorf("%s: Refs(ByFile %s): got %d index hits, want %d", us, file, c_refFileIndex_getByFile, want)
			}
		}
	}
}
