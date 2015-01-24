package store

import (
	"fmt"
	"reflect"
	"testing"

	"sourcegraph.com/sourcegraph/srclib/graph"
	"sourcegraph.com/sourcegraph/srclib/unit"
)

type labeledRepoStoreImporter struct {
	RepoStoreImporter
	label string
}

func (s *labeledRepoStoreImporter) String() string {
	return fmt.Sprintf("%s: %s", s.RepoStoreImporter, s.label)
}

func testRepoStore(t *testing.T, newFn func() RepoStoreImporter) {
	testRepoStore_uninitialized(t, &labeledRepoStoreImporter{newFn(), "uninitialized"})
	testRepoStore_Import_empty(t, &labeledRepoStoreImporter{newFn(), "import empty"})
	testRepoStore_Import(t, &labeledRepoStoreImporter{newFn(), "import"})
	testRepoStore_Version(t, &labeledRepoStoreImporter{newFn(), "version"})
	testRepoStore_Versions(t, &labeledRepoStoreImporter{newFn(), "versions"})
	testRepoStore_Unit(t, &labeledRepoStoreImporter{newFn(), "unit"})
	testRepoStore_Units(t, &labeledRepoStoreImporter{newFn(), "units"})
	testRepoStore_Def(t, &labeledRepoStoreImporter{newFn(), "def"})
	testRepoStore_Defs(t, &labeledRepoStoreImporter{newFn(), "defs"})
	testRepoStore_Defs_ByCommitID_ByFile(t, &labeledRepoStoreImporter{newFn(), "Defs(ByCommitID,ByFile)"})
	testRepoStore_Refs(t, &labeledRepoStoreImporter{newFn(), "refs"})
}

func testRepoStore_uninitialized(t *testing.T, rs RepoStoreImporter) {
	version, err := rs.Version(VersionKey{CommitID: "c"})
	if err == nil {
		t.Errorf("%s: Version: got nil err", rs)
	}
	if version != nil {
		t.Errorf("%s: Version: got version %v, want nil", rs, version)
	}

	versions, err := rs.Versions()
	if err == nil {
		t.Errorf("%s: Versions(): got nil err", rs)
	}
	if len(versions) != 0 {
		t.Errorf("%s: Versions(): got versions %v, want empty", rs, versions)
	}

	testTreeStore_uninitialized(t, rs)
}

func testRepoStore_Import_empty(t *testing.T, rs RepoStoreImporter) {
	if err := rs.Import("c", nil, graph.Output{}); err != nil {
		t.Errorf("%s: Import(c, nil, empty): %s", rs, err)
	}
	testTreeStore_empty(t, rs)
}

func testRepoStore_Import(t *testing.T, rs RepoStoreImporter) {
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
	if err := rs.Import("c", unit, data); err != nil {
		t.Errorf("%s: Import(c, %v, data): %s", rs, unit, err)
	}
}

func testRepoStore_Version(t *testing.T, rs RepoStoreImporter) {
	unit := &unit.SourceUnit{Type: "t", Name: "u"}
	if err := rs.Import("c", unit, graph.Output{}); err != nil {
		t.Errorf("%s: Import(c, %v, empty data): %s", rs, unit, err)
	}

	want := &Version{CommitID: "c"}

	version, err := rs.Version(VersionKey{CommitID: "c"})
	if err != nil {
		t.Errorf("%s: Version(c): %s", rs, err)
	}
	if !reflect.DeepEqual(version, want) {
		t.Errorf("%s: Version(c): got %v, want %v", rs, version, want)
	}
}

func testRepoStore_Versions(t *testing.T, rs RepoStoreImporter) {
	for _, version := range []string{"c1", "c2"} {
		unit := &unit.SourceUnit{Type: "t1", Name: "u1"}
		if err := rs.Import(version, unit, graph.Output{}); err != nil {
			t.Errorf("%s: Import(%s, %v, empty data): %s", rs, version, unit, err)
		}
	}

	want := []*Version{{CommitID: "c1"}, {CommitID: "c2"}}

	versions, err := rs.Versions()
	if err != nil {
		t.Errorf("%s: Versions(): %s", rs, err)
	}
	if !reflect.DeepEqual(versions, want) {
		t.Errorf("%s: Versions(): got %v, want %v", rs, versions, want)
	}
}

func testRepoStore_Unit(t *testing.T, rs RepoStoreImporter) {
	u := &unit.SourceUnit{Type: "t", Name: "u"}
	if err := rs.Import("c", u, graph.Output{}); err != nil {
		t.Errorf("%s: Import(c, %v, empty data): %s", rs, u, err)
	}

	want := &unit.SourceUnit{CommitID: "c", Type: "t", Name: "u"}

	key := unit.Key{CommitID: "c", UnitType: "t", Unit: "u"}
	unit, err := rs.Unit(key)
	if err != nil {
		t.Errorf("%s: Unit(%v): %s", rs, key, err)
	}
	if !reflect.DeepEqual(unit, want) {
		t.Errorf("%s: Unit(%v): got %v, want %v", rs, key, unit, want)
	}
}

func testRepoStore_Units(t *testing.T, rs RepoStoreImporter) {
	units := []*unit.SourceUnit{
		{Type: "t1", Name: "u1"},
		{Type: "t2", Name: "u2"},
	}
	for _, unit := range units {
		if err := rs.Import("c", unit, graph.Output{}); err != nil {
			t.Errorf("%s: Import(c, %v, empty data): %s", rs, unit, err)
		}
	}

	want := []*unit.SourceUnit{
		{CommitID: "c", Type: "t1", Name: "u1"},
		{CommitID: "c", Type: "t2", Name: "u2"},
	}

	units, err := rs.Units()
	if err != nil {
		t.Errorf("%s: Units(): %s", rs, err)
	}
	if !reflect.DeepEqual(units, want) {
		t.Errorf("%s: Units(): got %v, want %v", rs, units, want)
	}
}

func testRepoStore_Def(t *testing.T, rs RepoStoreImporter) {
	unit := &unit.SourceUnit{Type: "t", Name: "u"}
	data := graph.Output{
		Defs: []*graph.Def{
			{
				DefKey: graph.DefKey{Path: "p"},
				Name:   "n",
			},
		},
	}
	if err := rs.Import("c", unit, data); err != nil {
		t.Errorf("%s: Import(c, %v, data): %s", rs, unit, err)
	}

	def, err := rs.Def(graph.DefKey{Path: "p"})
	if !isInvalidKey(err) {
		t.Errorf("%s: Def(no unit): got err %v, want InvalidKeyError", rs, err)
	}
	if def != nil {
		t.Errorf("%s: Def: got def %v, want nil", rs, def)
	}

	def, err = rs.Def(graph.DefKey{UnitType: "t", Unit: "u", Path: "p"})
	if !isInvalidKey(err) {
		t.Errorf("%s: Def(no repo): got err %v, want InvalidKeyError", rs, err)
	}
	if def != nil {
		t.Errorf("%s: Def: got def %v, want nil", rs, def)
	}

	want := &graph.Def{
		DefKey: graph.DefKey{CommitID: "c", UnitType: "t", Unit: "u", Path: "p"},
		Name:   "n",
	}
	def, err = rs.Def(graph.DefKey{CommitID: "c", UnitType: "t", Unit: "u", Path: "p"})
	if err != nil {
		t.Errorf("%s: Def: %s", rs, err)
	}
	if !reflect.DeepEqual(def, want) {
		t.Errorf("%s: Def: got def %v, want %v", rs, def, want)
	}

	def2, err := rs.Def(graph.DefKey{CommitID: "c2", UnitType: "t", Unit: "u", Path: "p"})
	if !IsNotExist(err) {
		t.Errorf("%s: Def: got err %v, want IsNotExist-satisfying err", rs, err)
	}
	if def2 != nil {
		t.Errorf("%s: Def: got def %v, want nil", rs, def2)
	}
}

func testRepoStore_Defs(t *testing.T, rs RepoStoreImporter) {
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
	if err := rs.Import("c", unit, data); err != nil {
		t.Errorf("%s: Import(c, %v, data): %s", rs, unit, err)
	}

	want := []*graph.Def{
		{
			DefKey: graph.DefKey{CommitID: "c", UnitType: "t", Unit: "u", Path: "p1"},
			Name:   "n1",
		},
		{
			DefKey: graph.DefKey{CommitID: "c", UnitType: "t", Unit: "u", Path: "p2"},
			Name:   "n2",
		},
	}

	defs, err := rs.Defs()
	if err != nil {
		t.Errorf("%s: Defs(): %s", rs, err)
	}
	if !reflect.DeepEqual(defs, want) {
		t.Errorf("%s: Defs(): got defs %v, want %v", rs, defs, want)
	}
}

func testRepoStore_Defs_ByCommitID_ByFile(t *testing.T, rs RepoStoreImporter) {
	const numCommits = 2
	for c := 1; c <= numCommits; c++ {
		unit := &unit.SourceUnit{Type: "t", Name: "u", Files: []string{"f1", "f2"}}
		data := graph.Output{
			Defs: []*graph.Def{
				{DefKey: graph.DefKey{Path: "p1"}, File: "f1"},
				{DefKey: graph.DefKey{Path: "p2"}, File: "f2"},
			},
		}
		commitID := fmt.Sprintf("c%d", c)
		if err := rs.Import(commitID, unit, data); err != nil {
			t.Errorf("%s: Import(%s, %v, data): %s", rs, commitID, unit, err)
		}
	}

	want := []*graph.Def{
		{DefKey: graph.DefKey{CommitID: "c2", UnitType: "t", Unit: "u", Path: "p1"}, File: "f1"},
	}

	c_unitFilesIndex_getByPath = 0
	c_defFilesIndex_getByPath = 0
	defs, err := rs.Defs(ByCommitID("c2"), ByFiles("f1"))
	if err != nil {
		t.Fatalf("%s: Defs: %s", rs, err)
	}
	if !reflect.DeepEqual(defs, want) {
		t.Errorf("%s: Defs: got defs %v, want %v", rs, defs, want)
	}
	if isIndexedStore(rs) {
		if want := 1; c_unitFilesIndex_getByPath != want {
			t.Errorf("%s: Defs: got %d unitFilesIndex hits, want %d", rs, c_unitFilesIndex_getByPath, want)
		}
		if want := 1; c_defFilesIndex_getByPath != want {
			t.Errorf("%s: Defs: got %d defFilesIndex hits, want %d", rs, c_defFilesIndex_getByPath, want)
		}
	}
}

func testRepoStore_Refs(t *testing.T, rs RepoStoreImporter) {
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
	if err := rs.Import("c", unit, data); err != nil {
		t.Errorf("%s: Import(c, %v, data): %s", rs, unit, err)
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
			CommitID:    "c",
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
			CommitID:    "c",
		},
	}

	refs, err := rs.Refs()
	if err != nil {
		t.Errorf("%s: Refs(): %s", rs, err)
	}
	if !reflect.DeepEqual(refs, want) {
		t.Errorf("%s: Refs(): got refs %v, want %v", rs, refs, want)
	}
}
