package client

import (
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/sourcegraph/vcsstore/vcsclient"

	"strings"

	"sourcegraph.com/sourcegraph/Godeps/_workspace/src/github.com/kr/pretty"
	"sourcegraph.com/sourcegraph/api_router"
	"sourcegraph.com/sourcegraph/srcgraph/person"
	"sourcegraph.com/sourcegraph/srcgraph/repo"
)

func TestRepositoriesService_Get(t *testing.T) {
	setup()
	defer teardown()

	want := &Repository{Repository: &repo.Repository{RID: 1}}

	var called bool
	mux.HandleFunc(urlPath(t, api_router.Repository, map[string]string{"RepoURI": "r.com/x"}), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "GET")

		writeJSON(w, want)
	})

	repo_, _, err := client.Repositories.Get(RepositorySpec{URI: "r.com/x"}, nil)
	if err != nil {
		t.Errorf("Repositories.Get returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	if !reflect.DeepEqual(repo_, want) {
		t.Errorf("Repositories.Get returned %+v, want %+v", repo_, want)
	}
}

func TestRepositoriesService_GetOrCreate(t *testing.T) {
	setup()
	defer teardown()

	want := &Repository{Repository: &repo.Repository{RID: 1}}

	var called bool
	mux.HandleFunc(urlPath(t, api_router.RepositoriesGetOrCreate, map[string]string{"RepoURI": "r.com/x"}), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "PUT")

		writeJSON(w, want)
	})

	repo_, _, err := client.Repositories.GetOrCreate(RepositorySpec{URI: "r.com/x"}, nil)
	if err != nil {
		t.Errorf("Repositories.GetOrCreate returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	if !reflect.DeepEqual(repo_, want) {
		t.Errorf("Repositories.GetOrCreate returned %+v, want %+v", repo_, want)
	}
}

func TestRepositoriesService_Sync(t *testing.T) {
	setup()
	defer teardown()

	var called bool
	mux.HandleFunc(urlPath(t, api_router.RepositorySync, map[string]string{"RepoURI": "r.com/x"}), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "PUT")
	})

	_, err := client.Repositories.Sync("r.com/x")
	if err != nil {
		t.Errorf("Repositories.Sync returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}
}

func TestRepositoriesService_Create(t *testing.T) {
	setup()
	defer teardown()

	newRepo := NewRepositorySpec{Type: "git", CloneURLStr: "http://r.com/x"}
	want := &repo.Repository{RID: 1}

	var called bool
	mux.HandleFunc(urlPath(t, api_router.RepositoriesCreate, nil), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "POST")
		testBody(t, r, `{"Type":"git","CloneURL":"http://r.com/x"}`+"\n")

		writeJSON(w, want)
	})

	repo_, _, err := client.Repositories.Create(newRepo)
	if err != nil {
		t.Errorf("Repositories.Create returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	if !reflect.DeepEqual(repo_, want) {
		t.Errorf("Repositories.Create returned %+v, want %+v", repo_, want)
	}
}

func TestRepositoriesService_GetReadme(t *testing.T) {
	setup()
	defer teardown()

	want := &vcsclient.TreeEntry{Name: "hello"}
	want.ModTime = want.ModTime.In(time.UTC)

	var called bool
	mux.HandleFunc(urlPath(t, api_router.RepositoryReadme, map[string]string{"RepoURI": "r.com/x"}), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "GET")

		writeJSON(w, want)
	})

	readme, _, err := client.Repositories.GetReadme(RepositorySpec{URI: "r.com/x"})
	if err != nil {
		t.Errorf("Repositories.GetReadme returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	if !reflect.DeepEqual(readme, want) {
		t.Errorf("Repositories.GetReadme returned %+v, want %+v", readme, want)
	}
}

func TestRepositoriesService_List(t *testing.T) {
	setup()
	defer teardown()

	want := []*Repository{&Repository{Repository: &repo.Repository{RID: 1}}}

	var called bool
	mux.HandleFunc(urlPath(t, api_router.Repositories, nil), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "GET")
		testFormValues(t, r, values{
			"URIs":      "a,b",
			"Name":      "n",
			"Sort":      "name",
			"Direction": "asc",
			"NoFork":    "true",
			"PerPage":   "1",
			"Page":      "2",
		})

		writeJSON(w, want)
	})

	repos, _, err := client.Repositories.List(&RepositoryListOptions{
		URIs:        []string{"a", "b"},
		Name:        "n",
		Sort:        "name",
		Direction:   "asc",
		NoFork:      true,
		ListOptions: ListOptions{PerPage: 1, Page: 2},
	})
	if err != nil {
		t.Errorf("Repositories.List returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	if !reflect.DeepEqual(repos, want) {
		t.Errorf("Repositories.List returned %+v, want %+v with diff: %s", repos, want, strings.Join(pretty.Diff(want, repos), "\n"))
	}
}

func TestRepositoriesService_ListBadges(t *testing.T) {
	setup()
	defer teardown()

	want := []*Badge{{Name: "b"}}

	var called bool
	mux.HandleFunc(urlPath(t, api_router.RepositoryBadges, map[string]string{"RepoURI": "r.com/x"}), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "GET")

		writeJSON(w, want)
	})

	badges, _, err := client.Repositories.ListBadges(RepositorySpec{URI: "r.com/x"})
	if err != nil {
		t.Errorf("Repositories.ListBadges returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	if !reflect.DeepEqual(badges, want) {
		t.Errorf("Repositories.ListBadges returned %+v, want %+v", badges, want)
	}
}

func TestRepositoriesService_ListCounters(t *testing.T) {
	setup()
	defer teardown()

	want := []*Counter{{Name: "b"}}

	var called bool
	mux.HandleFunc(urlPath(t, api_router.RepositoryCounters, map[string]string{"RepoURI": "r.com/x"}), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "GET")

		writeJSON(w, want)
	})

	counters, _, err := client.Repositories.ListCounters(RepositorySpec{URI: "r.com/x"})
	if err != nil {
		t.Errorf("Repositories.ListCounters returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	if !reflect.DeepEqual(counters, want) {
		t.Errorf("Repositories.ListCounters returned %+v, want %+v", counters, want)
	}
}

func TestRepositoriesService_ListAuthors(t *testing.T) {
	setup()
	defer teardown()

	want := []*AugmentedRepoAuthor{{User: &person.User{Login: "b"}}}

	var called bool
	mux.HandleFunc(urlPath(t, api_router.RepositoryAuthors, map[string]string{"RepoURI": "r.com/x"}), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "GET")

		writeJSON(w, want)
	})

	authors, _, err := client.Repositories.ListAuthors(RepositorySpec{URI: "r.com/x"}, nil)
	if err != nil {
		t.Errorf("Repositories.ListAuthors returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	if !reflect.DeepEqual(authors, want) {
		t.Errorf("Repositories.ListAuthors returned %+v, want %+v", authors, want)
	}
}

func TestRepositoriesService_ListClients(t *testing.T) {
	setup()
	defer teardown()

	want := []*AugmentedRepoClient{{User: &person.User{Login: "b"}}}

	var called bool
	mux.HandleFunc(urlPath(t, api_router.RepositoryClients, map[string]string{"RepoURI": "r.com/x"}), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "GET")

		writeJSON(w, want)
	})

	clients, _, err := client.Repositories.ListClients(RepositorySpec{URI: "r.com/x"}, nil)
	if err != nil {
		t.Errorf("Repositories.ListClients returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	if !reflect.DeepEqual(clients, want) {
		t.Errorf("Repositories.ListClients returned %+v, want %+v", clients, want)
	}
}

func TestRepositoriesService_ListDependents(t *testing.T) {
	setup()
	defer teardown()

	want := []*AugmentedRepoDependent{{Repo: &repo.Repository{URI: "r2"}}}

	var called bool
	mux.HandleFunc(urlPath(t, api_router.RepositoryDependents, map[string]string{"RepoURI": "r.com/x"}), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "GET")

		writeJSON(w, want)
	})

	dependents, _, err := client.Repositories.ListDependents(RepositorySpec{URI: "r.com/x"}, nil)
	if err != nil {
		t.Errorf("Repositories.ListDependents returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	if !reflect.DeepEqual(dependents, want) {
		t.Errorf("Repositories.ListDependents returned %+v, want %+v", dependents, want)
	}
}

func TestRepositoriesService_ListDependencies(t *testing.T) {
	setup()
	defer teardown()

	want := []*AugmentedRepoDependency{{Repo: &repo.Repository{URI: "r2"}}}

	var called bool
	mux.HandleFunc(urlPath(t, api_router.RepositoryDependencies, map[string]string{"RepoURI": "r.com/x"}), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "GET")

		writeJSON(w, want)
	})

	dependencies, _, err := client.Repositories.ListDependencies(RepositorySpec{URI: "r.com/x"}, nil)
	if err != nil {
		t.Errorf("Repositories.ListDependencies returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	if !reflect.DeepEqual(dependencies, want) {
		t.Errorf("Repositories.ListDependencies returned %+v, want %+v", dependencies, want)
	}
}

func TestRepositoriesService_ListByOwner(t *testing.T) {
	setup()
	defer teardown()

	want := []*repo.Repository{{URI: "r.com/x"}}

	var called bool
	mux.HandleFunc(urlPath(t, api_router.PersonOwnedRepositories, map[string]string{"PersonSpec": "a"}), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "GET")

		writeJSON(w, want)
	})

	repos, _, err := client.Repositories.ListByOwner(PersonSpec{Login: "a"}, nil)
	if err != nil {
		t.Errorf("Repositories.ListByOwner returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	if !reflect.DeepEqual(repos, want) {
		t.Errorf("Repositories.ListByOwner returned %+v, want %+v", repos, want)
	}
}

func TestRepositoriesService_ListByContributor(t *testing.T) {
	setup()
	defer teardown()

	want := []*AugmentedRepoContribution{{Repo: &repo.Repository{URI: "r.com/x"}}}

	var called bool
	mux.HandleFunc(urlPath(t, api_router.PersonRepositoryContributions, map[string]string{"PersonSpec": "a"}), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "GET")
		testFormValues(t, r, values{"NoFork": "true"})

		writeJSON(w, want)
	})

	repos, _, err := client.Repositories.ListByContributor(PersonSpec{Login: "a"}, &RepositoryListByContributorOptions{NoFork: true})
	if err != nil {
		t.Errorf("Repositories.ListByContributor returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	if !reflect.DeepEqual(repos, want) {
		t.Errorf("Repositories.ListByContributor returned %+v, want %+v", repos, want)
	}
}

func TestRepositoriesService_ListByClient(t *testing.T) {
	setup()
	defer teardown()

	want := []*AugmentedRepoUsageByClient{{SymbolRepo: &repo.Repository{URI: "r.com/x"}}}

	var called bool
	mux.HandleFunc(urlPath(t, api_router.PersonRepositoryDependencies, map[string]string{"PersonSpec": "a"}), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "GET")

		writeJSON(w, want)
	})

	repos, _, err := client.Repositories.ListByClient(PersonSpec{Login: "a"}, nil)
	if err != nil {
		t.Errorf("Repositories.ListByClient returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	if !reflect.DeepEqual(repos, want) {
		t.Errorf("Repositories.ListByClient returned %+v, want %+v", repos, want)
	}
}

func TestRepositoriesService_ListByRefdAuthor(t *testing.T) {
	setup()
	defer teardown()

	want := []*AugmentedRepoUsageOfAuthor{{Repo: &repo.Repository{URI: "r.com/x"}}}

	var called bool
	mux.HandleFunc(urlPath(t, api_router.PersonRepositoryDependents, map[string]string{"PersonSpec": "a"}), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "GET")

		writeJSON(w, want)
	})

	repos, _, err := client.Repositories.ListByRefdAuthor(PersonSpec{Login: "a"}, nil)
	if err != nil {
		t.Errorf("Repositories.ListByRefdAuthor returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	if !reflect.DeepEqual(repos, want) {
		t.Errorf("Repositories.ListByRefdAuthor returned %+v, want %+v", repos, want)
	}
}
