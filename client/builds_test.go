package client

import (
	"net/http"
	"reflect"
	"testing"

	"sourcegraph.com/sourcegraph/api_router"
)

func TestRepositoryBuildsService_Get(t *testing.T) {
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
		t.Errorf("RepositoryBuilds.Get returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	normalizeBuildTime(build, want)
	if !reflect.DeepEqual(build, want) {
		t.Errorf("RepositoryBuilds.Get returned %+v, want %+v", build, want)
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
		t.Errorf("RepositoryBuilds.ListByRepository returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	normalizeBuildTime(builds...)
	normalizeBuildTime(want...)
	if !reflect.DeepEqual(builds, want) {
		t.Errorf("RepositoryBuilds.ListByRepository returned %+v, want %+v", builds, want)
	}
}

func normalizeBuildTime(bs ...*Build) {
	for _, b := range bs {
		normalizeTime(&b.CreatedAt)
		normalizeTime(&b.StartedAt.Time)
		normalizeTime(&b.EndedAt.Time)
	}
}
