package client

import (
	"net/http"
	"reflect"
	"testing"

	"sourcegraph.com/sourcegraph/api_router"
	"sourcegraph.com/sourcegraph/srcgraph/graph"
)

func TestSearchService_List(t *testing.T) {
	setup()
	defer teardown()

	want := []*Symbol{{Symbol: graph.Symbol{SID: 1}}}

	var called bool
	mux.HandleFunc(urlPath(t, api_router.Search, nil), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "GET")
		testFormValues(t, r, values{
			"Query":    "q",
			"Exported": "true",
			"Instant":  "true",
			"PerPage":  "1",
			"Page":     "2",
		})

		writeJSON(w, want)
	})

	people, _, err := client.Search.Search(&SearchOptions{
		Query:       "q",
		Exported:    true,
		Instant:     true,
		ListOptions: ListOptions{PerPage: 1, Page: 2},
	})
	if err != nil {
		t.Errorf("Search.Search returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	if !reflect.DeepEqual(people, want) {
		t.Errorf("Search.Search returned %+v, want %+v", people, want)
	}
}
