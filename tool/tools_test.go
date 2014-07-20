package tool

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestInstalledTools_List(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "srclib-toolchain-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	defer os.Setenv("PATH", os.Getenv("PATH"))
	os.Setenv("PATH", tmpdir)

	files := map[string]os.FileMode{"src-tool-foo": 0700, "src-tool-bar": 0600, "baz": 0700}
	for f, mode := range files {
		if err := ioutil.WriteFile(filepath.Join(tmpdir, f), nil, mode); err != nil {
			t.Fatal(err)
		}
	}

	tools, err := InstalledTools.List()
	if err != nil {
		t.Fatal(err)
	}

	got := toolNames(tools)
	want := []string{"foo"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got tools %v, want %v", got, want)
	}
}

func TestSrclibPathTools_List(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "srclib-toolchain-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	defer func(orig string) {
		SrclibPath = orig
	}(SrclibPath)
	SrclibPath = tmpdir

	files := map[string]struct{}{
		"a/a/Dockerfile": struct{}{}, "a/a/Srclibtool": struct{}{},
		"b/b/Dockerfile": struct{}{}, // no Srclibtool
		"c/c/Dockerfile": struct{}{}, // no Dockerfile
	}
	for f, _ := range files {
		if err := os.MkdirAll(filepath.Join(tmpdir, filepath.Dir(f)), 0700); err != nil {
			t.Fatal(err)
		}
		if err := ioutil.WriteFile(filepath.Join(tmpdir, f), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}

	tools, err := SrclibPathTools.List()
	if err != nil {
		t.Fatal(err)
	}

	got := toolNames(tools)
	want := []string{"a/a"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got tools %v, want %v", got, want)
	}
}

func toolNames(tools []Tool) []string {
	names := make([]string, len(tools))
	for i, tool := range tools {
		names[i] = tool.Name()
	}
	return names
}
