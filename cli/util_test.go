package cli

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/tools/godoc/vfs"
)

func TestReadJSONFileEmpty(t *testing.T) {
	f, err := ioutil.TempFile("", "read-json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	defer os.Remove(f.Name())
	err = readJSONFile(f.Name(), struct{}{})
	if err != errEmptyJSONFile {
		t.Errorf("Expected empty JSON file error, got %v", err)
	}
	fs := vfs.OS(filepath.Dir(f.Name()))
	err = readJSONFileFS(fs, filepath.Base(f.Name()), struct{}{})
	if err != errEmptyJSONFile {
		t.Errorf("Expected empty JSON file error (VFS), got %v", err)
	}
}

func TestReadJSONFileNotFound(t *testing.T) {
	err := readJSONFile("doesnotexist", struct{}{})
	if !os.IsNotExist(err) {
		t.Errorf("Expected does not exist error, got %v", err)
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	err = readJSONFileFS(vfs.OS(wd), "doesnotexist", struct{}{})
	if !os.IsNotExist(err) {
		t.Errorf("Expected does not exist error (VFS), got %v", err)
	}
}

func TestReadJSONFile(t *testing.T) {
	f, err := ioutil.TempFile("", "read-json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	defer os.Remove(f.Name())

	// Test valid decode.
	type msg struct{ A, B int }
	in := msg{5, 3}
	err = json.NewEncoder(f).Encode(in)
	if err != nil {
		t.Fatal(err)
	}
	out := new(msg)
	err = readJSONFile(f.Name(), out)
	if err != nil {
		t.Fatal(err)
	}
	if in != *out {
		t.Errorf("Read JSON file %+v, expected %+v", out, in)
	}
	fs := vfs.OS(filepath.Dir(f.Name()))
	outFS := new(msg)
	err = readJSONFileFS(fs, filepath.Base(f.Name()), outFS)
	if err != nil {
		t.Fatal(err)
	}
	if in != *outFS {
		t.Errorf("Read JSON file %+v (VFS), expected %+v", outFS, in)
	}

	// Test invalid decode.
	err = readJSONFile(f.Name(), &struct{ A, B bool }{})
	if err == nil {
		t.Error("Expected error because of invalid decode")
	}
	err = readJSONFileFS(fs, filepath.Base(f.Name()), &struct{ A, B bool }{})
	if err == nil {
		t.Error("Expected erorr because of invalid decode (VFS)")
	}
}
