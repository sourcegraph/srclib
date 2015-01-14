package graphstore_test

import (
	"io/ioutil"
	"log"
	"reflect"
	"testing"

	"sourcegraph.com/sourcegraph/srclib/graph"
	"sourcegraph.com/sourcegraph/srclib/graphstore"
)

var testStore *graphstore.Store

func init() {
	tmp, err := ioutil.TempDir("", "graphstore-test")
	if err != nil {
		log.Fatal(err)
	}
	testStore, err = graphstore.NewLocal(tmp)
	if err != nil {
		log.Fatal(err)
	}
}

func TestRefsStore(t *testing.T) {
	testDefKey := graph.DefKey{
		Repo:     "defrepo",
		UnitType: "defunittype",
		Unit:     "defunit",
		Path:     "defpath",
	}
	testRefs := []*graph.Ref{
		{
			DefRepo:     testDefKey.Repo,
			DefUnitType: testDefKey.UnitType,
			DefUnit:     testDefKey.Unit,
			DefPath:     testDefKey.Path,
			Repo:        "refrepo0",
			CommitID:    "ffffffffffffffffffffffffffffffffffffffff",
			UnitType:    "refunittype0",
			Unit:        "refunit0",
			File:        "reffile0",
			Start:       0,
			End:         3,
		},
		{
			DefRepo:     testDefKey.Repo,
			DefUnitType: testDefKey.UnitType,
			DefUnit:     testDefKey.Unit,
			DefPath:     testDefKey.Path,
			Repo:        "refrepo0",
			CommitID:    "ffffffffffffffffffffffffffffffffffffffff",
			UnitType:    "refunittype0",
			Unit:        "refunit0",
			File:        "reffile0",
			Start:       4,
			End:         8,
		},
		{
			DefRepo:     testDefKey.Repo,
			DefUnitType: testDefKey.UnitType,
			DefUnit:     testDefKey.Unit,
			DefPath:     testDefKey.Path,
			Repo:        "refrepo1",
			CommitID:    "rrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrr",
			UnitType:    "refunittype1",
			Unit:        "refunit1",
			File:        "reffile1",
			Start:       0,
			End:         3,
		},
	}
	if err := testStore.StoreRefs(testRefs); err != nil {
		t.Fatal(err)
	}
	// First, we list all of the refs for testDefKey.
	refs, err := testStore.ListRefs(testDefKey, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, r0 := range testRefs {
		var exist bool
		for _, r1 := range refs {
			if reflect.DeepEqual(r0, r1) {
				exist = true
				break
			}
		}
		if !exist {
			t.Errorf("%+v not found in %+v", r0, refs)
		}
	}
	// Now, we get the refs from only one repository.
	refs, err = testStore.ListRefs(testDefKey, &graphstore.ListRefsOptions{Repo: "refrepo1"})
	if err != nil {
		t.Fatal(err)
	}
	if wantLen := 1; len(refs) != wantLen {
		t.Fatalf("Wrong number of refs. Wanted %d, got %d.", wantLen, len(refs))
	}
	if wantRef := testRefs[2]; !reflect.DeepEqual(refs[0], wantRef) {
		t.Fatalf("Listed wrong ref. Wanted %+v, got %+v", wantRef, refs[0])
	}
}
