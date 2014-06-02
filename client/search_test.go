package client

import (
	"net/http"
	"reflect"
	"testing"

	"sourcegraph.com/sourcegraph/api_router"
)

func TestSearchService_Search(t *testing.T) {
	setup()
	defer teardown()

	want := &SearchResults{}

	var called bool
	mux.HandleFunc(urlPath(t, api_router.Search, nil), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "GET")
		testFormValues(t, r, values{
			"q":       "q",
			"PerPage": "1",
			"Page":    "2",
		})

		writeJSON(w, want)
	})

	results, _, err := client.Search.Search(&SearchOptions{
		Query:       "q",
		ListOptions: ListOptions{PerPage: 1, Page: 2},
	})
	if err != nil {
		t.Errorf("Search.Search returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	if !reflect.DeepEqual(results, want) {
		t.Errorf("Search.Search returned %+v, want %+v", results, want)
	}
}
