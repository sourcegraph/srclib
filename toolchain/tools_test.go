package toolchain

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestFindAllInPATH(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "srclib-toolchains-findallinpath")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	os.Setenv("PATH", tmpdir)
	defer os.Setenv("PATH", os.Getenv("PATH"))

	files := map[string]os.FileMode{"src-tool-foo": 0755, "src-tool-bar": 0644, "baz": 0755}
	for f, mode := range files {
		if err := ioutil.WriteFile(filepath.Join(tmpdir, f), nil, mode); err != nil {
			t.Fatal(err)
		}
	}

	ts, err := FindAllInPATH()
	if err != nil {
		t.Fatal(err)
	}

	want := []string{filepath.Join(tmpdir, "src-tool-foo")}
	if !reflect.DeepEqual(ts, want) {
		t.Errorf("got tools %v, want %v", ts, want)
	}
}
