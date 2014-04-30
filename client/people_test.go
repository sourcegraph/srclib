package client

import (
	"net/http"
	"reflect"
	"testing"

	"sourcegraph.com/sourcegraph/api_router"
	"sourcegraph.com/sourcegraph/srcgraph/person"
)

func TestPersonSpec(t *testing.T) {
	tests := []struct {
		str  string
		spec PersonSpec
	}{}

	for _, test := range tests {
		spec, err := ParsePersonSpec(test.str)
		if err != nil {
			t.Errorf("%q: ParsePersonSpec failed: %s", test.str, err)
			continue
		}
		if spec != test.spec {
			t.Errorf("%q: got spec %+v, want %+v", test.str, spec, test.spec)
			continue
		}

		str := test.spec.PathComponent()
		if str != test.str {
			t.Errorf("%+v: got str %q, want %q", test.spec, str, test.str)
			continue
		}
	}
}

func TestPeopleService_Get(t *testing.T) {
	setup()
	defer teardown()

	want := &person.User{UID: 1}

	var called bool
	mux.HandleFunc(urlPath(t, api_router.Person, map[string]string{"PersonSpec": "a"}), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "GET")

		writeJSON(w, want)
	})

	person_, _, err := client.People.Get(PersonSpec{Login: "a"})
	if err != nil {
		t.Errorf("People.Get returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	if !reflect.DeepEqual(person_, want) {
		t.Errorf("People.Get returned %+v, want %+v", person_, want)
	}
}

func TestPeopleService_GetOrCreateFromGitHub(t *testing.T) {
	setup()
	defer teardown()

	want := &person.User{UID: 1, Login: "a"}

	var called bool
	mux.HandleFunc(urlPath(t, api_router.PersonFromGitHub, map[string]string{"GitHubUserSpec": "a"}), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "GET")

		writeJSON(w, want)
	})

	person_, _, err := client.People.GetOrCreateFromGitHub(GitHubUserSpec{Login: "a"})
	if err != nil {
		t.Errorf("People.GetOrCreateFromGitHub returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	if !reflect.DeepEqual(person_, want) {
		t.Errorf("People.GetOrCreateFromGitHub returned %+v, want %+v", person_, want)
	}
}

func TestPeopleService_Sync(t *testing.T) {
	setup()
	defer teardown()

	var called bool
	mux.HandleFunc(urlPath(t, api_router.PersonSync, map[string]string{"PersonSpec": "a"}), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "PUT")
	})

	_, err := client.People.Sync(PersonSpec{Login: "a"})
	if err != nil {
		t.Errorf("People.Sync returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}
}

func TestPeopleService_List(t *testing.T) {
	setup()
	defer teardown()

	want := []*person.User{{UID: 1}}

	var called bool
	mux.HandleFunc(urlPath(t, api_router.People, nil), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "GET")
		testFormValues(t, r, values{
			"Query":     "q",
			"Sort":      "name",
			"Direction": "asc",
			"PerPage":   "1",
			"Page":      "2",
		})

		writeJSON(w, want)
	})

	people, _, err := client.People.List(&PersonListOptions{
		Query:       "q",
		Sort:        "name",
		Direction:   "asc",
		ListOptions: ListOptions{PerPage: 1, Page: 2},
	})
	if err != nil {
		t.Errorf("People.List returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	if !reflect.DeepEqual(people, want) {
		t.Errorf("People.List returned %+v, want %+v", people, want)
	}
}

func TestPeopleService_ListAuthors(t *testing.T) {
	setup()
	defer teardown()

	want := []*AugmentedPersonUsageByClient{{Author: &person.User{UID: 1}}}

	var called bool
	mux.HandleFunc(urlPath(t, api_router.PersonAuthors, map[string]string{"PersonSpec": "a"}), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "GET")

		writeJSON(w, want)
	})

	authors, _, err := client.People.ListAuthors(PersonSpec{Login: "a"}, nil)
	if err != nil {
		t.Errorf("People.ListAuthors returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	if !reflect.DeepEqual(authors, want) {
		t.Errorf("People.ListAuthors returned %+v, want %+v", authors, want)
	}
}

func TestPeopleService_ListClients(t *testing.T) {
	setup()
	defer teardown()

	want := []*AugmentedPersonUsageOfAuthor{{Client: &person.User{UID: 1}}}

	var called bool
	mux.HandleFunc(urlPath(t, api_router.PersonClients, map[string]string{"PersonSpec": "a"}), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "GET")

		writeJSON(w, want)
	})

	clients, _, err := client.People.ListClients(PersonSpec{Login: "a"}, nil)
	if err != nil {
		t.Errorf("People.ListClients returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	if !reflect.DeepEqual(clients, want) {
		t.Errorf("People.ListClients returned %+v, want %+v", clients, want)
	}
}
