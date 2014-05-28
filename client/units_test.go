package client

import (
	"net/http"
	"reflect"
	"testing"

	"sourcegraph.com/sourcegraph/api_router"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
)

func TestUnitsService_List(t *testing.T) {
	setup()
	defer teardown()

	want := []*unit.RepoSourceUnit{
		{
			Repo:     "r",
			UnitType: "t",
			Data:     []byte(`{}`),
		},
	}

	var called bool
	mux.HandleFunc(urlPath(t, api_router.Units, nil), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "GET")
		testFormValues(t, r, values{
			"RepositoryURI": "r1",
			"PerPage":       "1",
			"Page":          "2",
		})

		writeJSON(w, want)
	})

	units, _, err := client.Units.List(&UnitListOptions{
		RepositoryURI: "r1",
		ListOptions:   ListOptions{PerPage: 1, Page: 2},
	})
	if err != nil {
		t.Errorf("Units.List returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	if !reflect.DeepEqual(units, want) {
		t.Errorf("Units.List returned %+v, want %+v", units, want)
	}
}
