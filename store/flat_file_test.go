package store

import (
	"io/ioutil"
	"testing"

	"sourcegraph.com/sourcegraph/rwvfs"
)

func TestFlatFileUnitStore(t *testing.T) {
	testUnitStore(t, func() unitStoreImporter {
		tmpDir, err := ioutil.TempDir("", "srclib-TestFlatFileUnitStore")
		if err != nil {
			t.Fatal(err)
		}
		return &flatFileUnitStore{rwvfs.OS(tmpDir)}
	})
}

func TestFlatFileTreeStore(t *testing.T) {
	testTreeStore(t, func() treeStoreImporter {
		tmpDir, err := ioutil.TempDir("", "srclib-TestFlatFileTreeStore")
		if err != nil {
			t.Fatal(err)
		}
		fs := rwvfs.OS(tmpDir)
		setCreateParentDirs(fs)
		return newFlatFileTreeStore(rwvfs.Sub(fs, "tree"))
	})
}

func TestFlatFileRepoStore(t *testing.T) {
	testRepoStore(t, func() repoStoreImporter {
		tmpDir, err := ioutil.TempDir("", "srclib-TestFlatFileRepoStore")
		if err != nil {
			t.Fatal(err)
		}
		fs := rwvfs.OS(tmpDir)
		setCreateParentDirs(fs)
		return newFlatFileRepoStore(rwvfs.Sub(fs, "repo"))
	})
}

func setCreateParentDirs(fs rwvfs.FileSystem) {
	type createParents interface {
		CreateParentDirs(bool)
	}
	if fs, ok := fs.(createParents); ok {
		fs.CreateParentDirs(true)
	}
}
