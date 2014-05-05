package client

import (
	"net/http"
	"reflect"
	"testing"

	"sourcegraph.com/sourcegraph/api_router"
	"sourcegraph.com/sourcegraph/srcgraph/graph"
	"sourcegraph.com/sourcegraph/srcgraph/person"
	"sourcegraph.com/sourcegraph/srcgraph/repo"
)

func TestSymbolsService_Get(t *testing.T) {
	setup()
	defer teardown()

	want := &Symbol{Symbol: graph.Symbol{SID: 1}}

	var called bool
	mux.HandleFunc(urlPath(t, api_router.Symbol, map[string]string{"RepoURI": "r.com/x", "UnitType": "t", "Unit": "u", "Path": "p"}), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "GET")
		testFormValues(t, r, values{"Annotate": "true"})

		writeJSON(w, want)
	})

	repo_, _, err := client.Symbols.Get(SymbolSpec{Repo: "r.com/x", UnitType: "t", Unit: "u", Path: "p"}, &SymbolGetOptions{Annotate: true})
	if err != nil {
		t.Errorf("Symbols.Get returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	if !reflect.DeepEqual(repo_, want) {
		t.Errorf("Symbols.Get returned %+v, want %+v", repo_, want)
	}
}

func TestSymbolsService_List(t *testing.T) {
	setup()
	defer teardown()

	want := []*Symbol{{Symbol: graph.Symbol{SID: 1}}}

	var called bool
	mux.HandleFunc(urlPath(t, api_router.Symbols, nil), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "GET")
		testFormValues(t, r, values{
			"RepositoryURI": "r1",
			"Query":         "q",
			"Sort":          "name",
			"Direction":     "asc",
			"Kinds":         "a,b",
			"SpecificKind":  "k",
			"Exported":      "true",
			"Doc":           "true",
			"PerPage":       "1",
			"Page":          "2",
		})

		writeJSON(w, want)
	})

	symbols, _, err := client.Symbols.List(&SymbolListOptions{
		RepositoryURI: "r1",
		Query:         "q",
		Sort:          "name",
		Direction:     "asc",
		Kinds:         []string{"a", "b"},
		SpecificKind:  "k",
		Exported:      true,
		Doc:           true,
		ListOptions:   ListOptions{PerPage: 1, Page: 2},
	})
	if err != nil {
		t.Errorf("Symbols.List returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	if !reflect.DeepEqual(symbols, want) {
		t.Errorf("Symbols.List returned %+v, want %+v", symbols, want)
	}
}

func TestSymbolsService_ListExamples(t *testing.T) {
	setup()
	defer teardown()

	want := []*Example{{Ref: graph.Ref{File: "f"}}}

	var called bool
	mux.HandleFunc(urlPath(t, api_router.SymbolExamples, map[string]string{"RepoURI": "r.com/x", "UnitType": "t", "Unit": "u", "Path": "p"}), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "GET")

		writeJSON(w, want)
	})

	refs, _, err := client.Symbols.ListExamples(SymbolSpec{Repo: "r.com/x", UnitType: "t", Unit: "u", Path: "p"}, nil)
	if err != nil {
		t.Errorf("Symbols.ListExamples returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	if !reflect.DeepEqual(refs, want) {
		t.Errorf("Symbols.ListExamples returned %+v, want %+v", refs, want)
	}
}

func TestSymbolsService_ListAuthors(t *testing.T) {
	setup()
	defer teardown()

	want := []*AugmentedSymbolAuthor{{User: &person.User{Login: "b"}}}

	var called bool
	mux.HandleFunc(urlPath(t, api_router.SymbolAuthors, map[string]string{"RepoURI": "r.com/x", "UnitType": "t", "Unit": "u", "Path": "p"}), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "GET")

		writeJSON(w, want)
	})

	authors, _, err := client.Symbols.ListAuthors(SymbolSpec{Repo: "r.com/x", UnitType: "t", Unit: "u", Path: "p"}, nil)
	if err != nil {
		t.Errorf("Symbols.ListAuthors returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	if !reflect.DeepEqual(authors, want) {
		t.Errorf("Symbols.ListAuthors returned %+v, want %+v", authors, want)
	}
}

func TestSymbolsService_ListClients(t *testing.T) {
	setup()
	defer teardown()

	want := []*AugmentedSymbolClient{{User: &person.User{Login: "b"}}}

	var called bool
	mux.HandleFunc(urlPath(t, api_router.SymbolClients, map[string]string{"RepoURI": "r.com/x", "UnitType": "t", "Unit": "u", "Path": "p"}), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "GET")

		writeJSON(w, want)
	})

	clients, _, err := client.Symbols.ListClients(SymbolSpec{Repo: "r.com/x", UnitType: "t", Unit: "u", Path: "p"}, nil)
	if err != nil {
		t.Errorf("Symbols.ListClients returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	if !reflect.DeepEqual(clients, want) {
		t.Errorf("Symbols.ListClients returned %+v, want %+v", clients, want)
	}
}

func TestSymbolsService_ListDependents(t *testing.T) {
	setup()
	defer teardown()

	want := []*AugmentedSymbolDependent{{Repo: &repo.Repository{URI: "r2"}}}

	var called bool
	mux.HandleFunc(urlPath(t, api_router.SymbolDependents, map[string]string{"RepoURI": "r.com/x", "UnitType": "t", "Unit": "u", "Path": "p"}), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "GET")

		writeJSON(w, want)
	})

	dependents, _, err := client.Symbols.ListDependents(SymbolSpec{Repo: "r.com/x", UnitType: "t", Unit: "u", Path: "p"}, nil)
	if err != nil {
		t.Errorf("Symbols.ListDependents returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	if !reflect.DeepEqual(dependents, want) {
		t.Errorf("Symbols.ListDependents returned %+v, want %+v", dependents, want)
	}
}

func TestSymbolsService_ListImplementations(t *testing.T) {
	setup()
	defer teardown()

	want := []*Symbol{{Symbol: graph.Symbol{SID: 1}}}

	var called bool
	mux.HandleFunc(urlPath(t, api_router.SymbolImplementations, map[string]string{"RepoURI": "r.com/x", "UnitType": "t", "Unit": "u", "Path": "p"}), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "GET")

		writeJSON(w, want)
	})

	implementations, _, err := client.Symbols.ListImplementations(SymbolSpec{Repo: "r.com/x", UnitType: "t", Unit: "u", Path: "p"}, nil)
	if err != nil {
		t.Errorf("Symbols.ListImplementations returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	if !reflect.DeepEqual(implementations, want) {
		t.Errorf("Symbols.ListImplementations returned %+v, want %+v", implementations, want)
	}
}

func TestSymbolsService_ListInterfaces(t *testing.T) {
	setup()
	defer teardown()

	want := []*Symbol{{Symbol: graph.Symbol{SID: 1}}}

	var called bool
	mux.HandleFunc(urlPath(t, api_router.SymbolInterfaces, map[string]string{"RepoURI": "r.com/x", "UnitType": "t", "Unit": "u", "Path": "p"}), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "GET")

		writeJSON(w, want)
	})

	interfaces, _, err := client.Symbols.ListInterfaces(SymbolSpec{Repo: "r.com/x", UnitType: "t", Unit: "u", Path: "p"}, nil)
	if err != nil {
		t.Errorf("Symbols.ListInterfaces returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	if !reflect.DeepEqual(interfaces, want) {
		t.Errorf("Symbols.ListInterfaces returned %+v, want %+v", interfaces, want)
	}
}

func TestSymbolsService_CountByRepository(t *testing.T) {
	setup()
	defer teardown()

	want := &graph.SymbolCounts{Exported: 1}

	var called bool
	mux.HandleFunc(urlPath(t, api_router.RepositorySymbolCounts, map[string]string{"RepoURI": "r.com/x"}), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "GET")

		writeJSON(w, want)
	})

	counts, _, err := client.Symbols.CountByRepository(RepositorySpec{URI: "r.com/x"})
	if err != nil {
		t.Errorf("Symbols.CountByRepository returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	if !reflect.DeepEqual(counts, want) {
		t.Errorf("Symbols.CountByRepository returned %+v, want %+v", counts, want)
	}
}
