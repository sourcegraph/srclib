package client

import (
	"net/http"
	"reflect"
	"testing"

	"sourcegraph.com/sourcegraph/api_router"
)

func TestBuildsService_Get(t *testing.T) {
	setup()
	defer teardown()

	want := &Build{BID: 1}

	var called bool
	mux.HandleFunc(urlPath(t, api_router.RepositoryBuild, map[string]string{"RepoURI": "r.com/x", "BID": "1"}), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "GET")

		writeJSON(w, want)
	})

	build, _, err := client.Builds.Get(BuildSpec{Repo: RepositorySpec{URI: "r.com/x"}, BID: 1}, nil)
	if err != nil {
		t.Errorf("Builds.Get returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	normalizeBuildTime(build, want)
	if !reflect.DeepEqual(build, want) {
		t.Errorf("Builds.Get returned %+v, want %+v", build, want)
	}
}

func TestBuildsService_List(t *testing.T) {
	setup()
	defer teardown()

	want := []*Build{{BID: 1}}

	var called bool
	mux.HandleFunc(urlPath(t, api_router.Builds, nil), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "GET")

		writeJSON(w, want)
	})

	builds, _, err := client.Builds.List(nil)
	if err != nil {
		t.Errorf("Builds.List returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	normalizeBuildTime(builds...)
	normalizeBuildTime(want...)
	if !reflect.DeepEqual(builds, want) {
		t.Errorf("Builds.List returned %+v, want %+v", builds, want)
	}
}

func TestRepositoryBuildsService_ListByRepository(t *testing.T) {
	setup()
	defer teardown()

	want := []*Build{{BID: 1}}

	var called bool
	mux.HandleFunc(urlPath(t, api_router.RepositoryBuilds, map[string]string{"RepoURI": "r.com/x"}), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "GET")

		writeJSON(w, want)
	})

	builds, _, err := client.Builds.ListByRepository(RepositorySpec{URI: "r.com/x"}, nil)
	if err != nil {
		t.Errorf("Builds.ListByRepository returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	normalizeBuildTime(builds...)
	normalizeBuildTime(want...)
	if !reflect.DeepEqual(builds, want) {
		t.Errorf("Builds.ListByRepository returned %+v, want %+v", builds, want)
	}
}

func TestBuildsService_Create(t *testing.T) {
	setup()
	defer teardown()

	config := BuildConfig{Import: true, Queue: true}
	want := &Build{BID: 123, Repo: 456}

	var called bool
	mux.HandleFunc(urlPath(t, api_router.RepositoryBuildsCreate, map[string]string{"RepoURI": "r.com/x"}), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "POST")
		testBody(t, r, `{"Import":true,"Queue":true}`+"\n")

		writeJSON(w, want)
	})

	build_, _, err := client.Builds.Create(RepositorySpec{URI: "r.com/x"}, config)
	if err != nil {
		t.Errorf("Builds.Create returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	normalizeBuildTime(build_)
	normalizeBuildTime(want)
	if !reflect.DeepEqual(build_, want) {
		t.Errorf("Builds.Create returned %+v, want %+v", build_, want)
	}
}

func normalizeBuildTime(bs ...*Build) {
	for _, b := range bs {
		normalizeTime(&b.CreatedAt)
		normalizeTime(&b.StartedAt.Time)
		normalizeTime(&b.EndedAt.Time)
	}
}
