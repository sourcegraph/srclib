package client

import (
	"html/template"
	"io"
	"net/http"
	"reflect"
	"testing"

	"sourcegraph.com/sourcegraph/api_router"
)

func TestRepositoryTreeService_Get(t *testing.T) {
	setup()
	defer teardown()

	want := &TreeEntry{Data: template.HTML("hello"), Type: File}

	var called bool
	mux.HandleFunc(urlPath(t, api_router.RepositoryTreeEntry, map[string]string{"RepoURI": "r.com/x", "Rev": "v", "Path": "p"}), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "GET")
		testFormValues(t, r, values{"Annotated": "true"})

		io.WriteString(w, string(want.Data))
	})

	data, _, err := client.RepositoryTree.Get(TreeEntrySpec{
		Repo: RepositorySpec{URI: "r.com/x"},
		Rev:  "v",
		Path: "p",
	}, &RepositoryTreeGetOptions{Annotated: true})
	if err != nil {
		t.Errorf("RepositoryTree.Get returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	if !reflect.DeepEqual(data, want) {
		t.Errorf("RepositoryTree.Get returned %+v, want %+v", data, want)
	}
}

func TestRepositoryTreeService_Get_file(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc(urlPath(t, api_router.RepositoryTreeEntry, map[string]string{"RepoURI": "r.com/x", "Rev": "v", "Path": "p"}), func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("content-type", "text/plain")
	})

	entry, _, err := client.RepositoryTree.Get(TreeEntrySpec{
		Repo: RepositorySpec{URI: "r.com/x"},
		Rev:  "v",
		Path: "p",
	}, &RepositoryTreeGetOptions{Annotated: true})
	if err != nil {
		t.Errorf("RepositoryTree.Get returned error: %v", err)
	}

	if entry.Type != File {
		t.Errorf("RepositoryTree.Get returned Type %q, want %q", entry.Type, File)
	}
}

func TestRepositoryTreeService_Get_directory(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc(urlPath(t, api_router.RepositoryTreeEntry, map[string]string{"RepoURI": "r.com/x", "Rev": "v", "Path": "p"}), func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("content-type", "application/x-directory")
	})

	entry, _, err := client.RepositoryTree.Get(TreeEntrySpec{
		Repo: RepositorySpec{URI: "r.com/x"},
		Rev:  "v",
		Path: "p",
	}, &RepositoryTreeGetOptions{Annotated: true})
	if err != nil {
		t.Errorf("RepositoryTree.Get returned error: %v", err)
	}

	if entry.Type != Dir {
		t.Errorf("RepositoryTree.Get returned Type %q, want %q", entry.Type, File)
	}
}
