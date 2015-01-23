package store

import (
	"fmt"
	"reflect"
	"testing"

	"sort"

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

func isIndexedStore(s interface{}) bool {
	switch s := s.(type) {
	case *labeledUnitStoreImporter:
		return isIndexedStore(s.unitStoreImporter)
	case *labeledTreeStoreImporter:
		return isIndexedStore(s.treeStoreImporter)
	case *labeledRepoStoreImporter:
		return isIndexedStore(s.RepoStoreImporter)
	case *labeledMultiRepoStoreImporter:
		return isIndexedStore(s.MultiRepoStoreImporter)
	case *indexedTreeStore:
		return true
	case *indexedUnitStore:
		return true
	case *flatFileRepoStore:
		return useIndexedStore
	default:
		return false
	}
}

func testTreeStore(t *testing.T, newFn func() treeStoreImporter) {
	testTreeStore_uninitialized(t, &labeledTreeStoreImporter{newFn(), "uninitialized"})
	testTreeStore_Import_empty(t, &labeledTreeStoreImporter{newFn(), "import empty"})
	testTreeStore_Import(t, &labeledTreeStoreImporter{newFn(), "import"})
	testTreeStore_Unit(t, &labeledTreeStoreImporter{newFn(), "unit"})
	testTreeStore_Units(t, &labeledTreeStoreImporter{newFn(), "unit"})
	testTreeStore_Units_ByFile(t, &labeledTreeStoreImporter{newFn(), "units by file"})
	testTreeStore_Def(t, &labeledTreeStoreImporter{newFn(), "def"})
	testTreeStore_Defs(t, &labeledTreeStoreImporter{newFn(), "defs"})
	testTreeStore_Defs_ByUnits(t, &labeledTreeStoreImporter{newFn(), "defs by units"})
	testTreeStore_Defs_ByFiles(t, &labeledTreeStoreImporter{newFn(), "defs by files"})
	testTreeStore_Refs(t, &labeledTreeStoreImporter{newFn(), "refs"})
	testTreeStore_Refs_ByFiles(t, &labeledTreeStoreImporter{newFn(), "refs by file"})
}

func testTreeStore_uninitialized(t *testing.T, ts TreeStore) {
	unit, err := ts.Unit(unit.Key{Repo: "r", CommitID: "c", UnitType: "t", Unit: "u"})
	if err == nil {
		t.Errorf("%s: Unit: got nil err", ts)
	}
	if unit != nil {
		t.Errorf("%s: Unit: got unit %v, want nil", ts, unit)
	}

	units, err := ts.Units()
	if err == nil {
		t.Errorf("%s: Units(): got nil err", ts)
	}
	if len(units) != 0 {
		t.Errorf("%s: Units(): got units %v, want empty", ts, units)
	}

	testUnitStore_uninitialized(t, ts)
}

func testTreeStore_empty(t *testing.T, ts TreeStore) {
	unit, err := ts.Unit(unit.Key{Repo: "r", CommitID: "c", UnitType: "t", Unit: "u"})
	if !IsNotExist(err) {
		t.Errorf("%s: Unit: got err %v, want IsNotExist-satisfying err", ts, err)
	}
	if unit != nil {
		t.Errorf("%s: Unit: got unit %v, want nil", ts, unit)
	}

	units, err := ts.Units()
	if err != nil {
		t.Errorf("%s: Units(): %s", ts, err)
	}
	if len(units) != 0 {
		t.Errorf("%s: Units(): got units %v, want empty", ts, units)
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
	unit := &unit.SourceUnit{Type: "t", Name: "u", Files: []string{"f"}}
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

	key := unit.Key{CommitID: "c", UnitType: "t", Unit: "u"}
	unit, err := ts.Unit(key)
	if err != nil {
		t.Errorf("%s: Unit(%v): %s", ts, key, err)
	}
	if !reflect.DeepEqual(unit, want) {
		t.Errorf("%s: Unit(%v): got %v, want %v", ts, key, unit, want)
	}
}

func testTreeStore_Units(t *testing.T, ts treeStoreImporter) {
	want := []*unit.SourceUnit{
		{Type: "t1", Name: "u1"},
		{Type: "t2", Name: "u2"},
		{Type: "t3", Name: "u3"},
	}
	for _, unit := range want {
		if err := ts.Import(unit, graph.Output{}); err != nil {
			t.Errorf("%s: Import(%v, empty data): %s", ts, unit, err)
		}
	}

	units, err := ts.Units()
	if err != nil {
		t.Errorf("%s: Units(): %s", ts, err)
	}
	if !reflect.DeepEqual(units, want) {
		t.Errorf("%s: Units(): got %v, want %v", ts, units, want)
	}

	units2, err := ts.Units(ByUnits(unit.ID2{Type: "t3", Name: "u3"}, unit.ID2{Type: "t1", Name: "u1"}))
	if err != nil {
		t.Errorf("%s: Units(3 and 1): %s", ts, err)
	}
	want2 := []*unit.SourceUnit{
		{Type: "t1", Name: "u1"},
		{Type: "t3", Name: "u3"},
	}
	sort.Sort(unit.SourceUnits(units2))
	sort.Sort(unit.SourceUnits(want2))
	if !reflect.DeepEqual(units2, want2) {
		t.Errorf("%s: Units(3 and 1): got %v, want %v", ts, units2, want2)
	}
}

func testTreeStore_Units_ByFile(t *testing.T, ts treeStoreImporter) {
	want := []*unit.SourceUnit{
		{Type: "t1", Name: "u1", Files: []string{"f1"}},
		{Type: "t2", Name: "u2", Files: []string{"f1", "f2"}},
		{Type: "t3", Name: "u3", Files: []string{"f1", "f3"}},
	}
	for _, unit := range want {
		if err := ts.Import(unit, graph.Output{}); err != nil {
			t.Errorf("%s: Import(%v, empty data): %s", ts, unit, err)
		}
	}

	c_unitFilesIndex_getByPath = 0
	units, err := ts.Units(ByFiles("f1"))
	if err != nil {
		t.Errorf("%s: Units(ByFiles f1): %s", ts, err)
	}
	if !reflect.DeepEqual(units, want) {
		t.Errorf("%s: Units(ByFiles f1): got %v, want %v", ts, units, want)
	}
	if isIndexedStore(ts) {
		if want := 1; c_unitFilesIndex_getByPath != want {
			t.Errorf("%s: Units(ByFiles f1): got %d index hits, want %d", ts, c_unitFilesIndex_getByPath, want)
		}
	}

	c_unitFilesIndex_getByPath = 0
	units2, err := ts.Units(ByFiles("f2"))
	if err != nil {
		t.Errorf("%s: Units(ByFiles f2): %s", ts, err)
	}
	want2 := []*unit.SourceUnit{
		{Type: "t2", Name: "u2", Files: []string{"f1", "f2"}},
	}
	if !reflect.DeepEqual(units2, want2) {
		t.Errorf("%s: Units(ByFiles f2): got %v, want %v", ts, units2, want2)
	}
	if isIndexedStore(ts) {
		if want := 1; c_unitFilesIndex_getByPath != want {
			t.Errorf("%s: Units(ByFiles f1): got %d index hits, want %d", ts, c_unitFilesIndex_getByPath, want)
		}
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
	if !isInvalidKey(err) {
		t.Errorf("%s: Def(no unit): got err %v, want InvalidKeyError", ts, err)
	}
	if def != nil {
		t.Errorf("%s: Def: got def %v, want nil", ts, def)
	}

	want := &graph.Def{
		DefKey: graph.DefKey{UnitType: "t", Unit: "u", Path: "p"},
		Name:   "n",
	}
	def, err = ts.Def(graph.DefKey{UnitType: "t", Unit: "u", Path: "p"})
	if err != nil {
		t.Errorf("%s: Def: %s", ts, err)
	}
	if !reflect.DeepEqual(def, want) {
		t.Errorf("%s: Def: got def %v, want %v", ts, def, want)
	}

	def2, err := ts.Def(graph.DefKey{UnitType: "t2", Unit: "u2", Path: "p"})
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

	defs, err := ts.Defs()
	if err != nil {
		t.Errorf("%s: Defs(): %s", ts, err)
	}
	if !reflect.DeepEqual(defs, want) {
		t.Errorf("%s: Defs(): got defs %v, want %v", ts, defs, want)
	}
}

func testTreeStore_Defs_ByUnits(t *testing.T, ts treeStoreImporter) {
	units := []*unit.SourceUnit{
		{Type: "t1", Name: "u1"},
		{Type: "t2", Name: "u2"},
		{Type: "t3", Name: "u3"},
	}
	for i, unit := range units {
		data := graph.Output{
			Defs: []*graph.Def{{DefKey: graph.DefKey{Path: fmt.Sprintf("p%d", i+1)}}},
		}
		if err := ts.Import(unit, data); err != nil {
			t.Errorf("%s: Import(%v, data): %s", ts, unit, err)
		}
	}

	want := []*graph.Def{
		{DefKey: graph.DefKey{UnitType: "t1", Unit: "u1", Path: "p1"}},
		{DefKey: graph.DefKey{UnitType: "t3", Unit: "u3", Path: "p3"}},
	}

	defs, err := ts.Defs(ByUnits(unit.ID2{Type: "t3", Name: "u3"}, unit.ID2{Type: "t1", Name: "u1"}))
	if err != nil {
		t.Errorf("%s: Defs(ByUnits): %s", ts, err)
	}
	sort.Sort(graph.Defs(defs))
	sort.Sort(graph.Defs(want))
	if !reflect.DeepEqual(defs, want) {
		t.Errorf("%s: Defs(ByUnits): got defs %v, want %v", ts, defs, want)
	}
}

func testTreeStore_Defs_ByFiles(t *testing.T, ts treeStoreImporter) {
	units := []*unit.SourceUnit{
		{Type: "t1", Name: "u1", Files: []string{"f1"}},
		{Type: "t2", Name: "u2", Files: []string{"f2"}},
	}
	for i, unit := range units {
		data := graph.Output{
			Defs: []*graph.Def{{DefKey: graph.DefKey{Path: fmt.Sprintf("p%d", i+1)}, File: fmt.Sprintf("f%d", i+1)}},
		}
		if err := ts.Import(unit, data); err != nil {
			t.Errorf("%s: Import(%v, data): %s", ts, unit, err)
		}
	}

	want := []*graph.Def{
		{DefKey: graph.DefKey{UnitType: "t2", Unit: "u2", Path: "p2"}, File: "f2"},
	}

	c_unitFilesIndex_getByPath = 0
	defs, err := ts.Defs(ByFiles("f2"))
	if err != nil {
		t.Errorf("%s: Defs(ByFiles f2): %s", ts, err)
	}
	if !reflect.DeepEqual(defs, want) {
		t.Errorf("%s: Defs(ByFiles f2): got defs %v, want %v", ts, defs, want)
	}
	if isIndexedStore(ts) {
		if want := 1; c_unitFilesIndex_getByPath != want {
			t.Errorf("%s: Defs(ByFiles f2): got %d index hits, want %d", ts, c_unitFilesIndex_getByPath, want)
		}
	}
}

func testTreeStore_Refs(t *testing.T, ts treeStoreImporter) {
	unit := &unit.SourceUnit{Type: "t", Name: "u", Files: []string{"f1", "f2"}}
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
			DefUnitType: "t",
			DefUnit:     "u",
			DefPath:     "p1",
			File:        "f1",
			Start:       1,
			End:         2,
			UnitType:    "t",
			Unit:        "u",
		},
		{
			DefUnitType: "t",
			DefUnit:     "u",
			DefPath:     "p2",
			File:        "f2",
			Start:       2,
			End:         3,
			UnitType:    "t",
			Unit:        "u",
		},
	}

	refs, err := ts.Refs()
	if err != nil {
		t.Errorf("%s: Refs(): %s", ts, err)
	}
	if !reflect.DeepEqual(refs, want) {
		t.Errorf("%s: Refs(): got refs %v, want %v", ts, refs, want)
	}
}

func testTreeStore_Refs_ByFiles(t *testing.T, ts treeStoreImporter) {
	refsByUnitByFile := map[string]map[string][]*graph.Ref{
		"u1": {
			"f1": {
				{DefPath: "p1", Start: 0, End: 5},
			},
			"f2": {
				{DefPath: "p1", Start: 0, End: 5},
				{DefPath: "p2", Start: 5, End: 10},
			},
		},
		"u2": {
			"f1": {
				{DefPath: "p1", Start: 5, End: 10},
			},
		},
	}
	refsByFile := map[string][]*graph.Ref{}
	for unitName, refsByFile0 := range refsByUnitByFile {
		u := &unit.SourceUnit{Type: "t", Name: unitName}
		var data graph.Output
		for file, refs := range refsByFile0 {
			u.Files = append(u.Files, file)
			for _, ref := range refs {
				ref.File = file
			}
			data.Refs = append(data.Refs, refs...)
			refsByFile[file] = append(refsByFile[file], refs...)
		}
		if err := ts.Import(u, data); err != nil {
			t.Errorf("%s: Import(%v, data): %s", ts, u, err)
		}
	}

	for file, wantRefs := range refsByFile {
		c_unitStores_Refs_last_numUnitsQueried = 0
		c_refFileIndex_getByFile = 0
		refs, err := ts.Refs(ByFiles(file))
		if err != nil {
			t.Fatalf("%s: Refs(ByFiles %s): %s", ts, file, err)
		}

		distinctRefUnits := map[string]struct{}{}
		for _, ref := range refs {
			distinctRefUnits[ref.Unit] = struct{}{}
		}

		// for test equality
		sort.Sort(refsByFileStartEnd(refs))
		sort.Sort(refsByFileStartEnd(wantRefs))
		cleanForImport(&graph.Output{Refs: refs}, "", "t", "u1")
		cleanForImport(&graph.Output{Refs: refs}, "", "t", "u2")

		if want := wantRefs; !reflect.DeepEqual(refs, want) {
			t.Errorf("%s: Refs(ByFiles %s): got refs %v, want %v", ts, file, refs, want)
		}
		if isIndexedStore(ts) {
			if want := len(distinctRefUnits); c_refFileIndex_getByFile != want {
				t.Errorf("%s: Refs(ByFiles %s): got %d index hits, want %d", ts, file, c_refFileIndex_getByFile, want)
			}
			if want := len(distinctRefUnits); c_unitStores_Refs_last_numUnitsQueried != want {
				t.Errorf("%s: Refs(ByFiles %s): got %d units queried, want %d", ts, file, c_unitStores_Refs_last_numUnitsQueried, want)
			}
		}
	}
}
