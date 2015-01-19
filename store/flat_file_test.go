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
		return &flatFileUnitStore{fs: rwvfs.OS(tmpDir), codec: JSONCodec{}}
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
		return newFlatFileTreeStore(rwvfs.Sub(fs, "tree"), nil)
	})
}

func TestFlatFileRepoStore(t *testing.T) {
	testRepoStore(t, func() RepoStoreImporter {
		tmpDir, err := ioutil.TempDir("", "srclib-TestFlatFileRepoStore")
		if err != nil {
			t.Fatal(err)
		}
		fs := rwvfs.OS(tmpDir)
		setCreateParentDirs(fs)
		return NewFlatFileRepoStore(rwvfs.Sub(fs, "repo"), nil)
	})
}

func TestFlatFileMultiRepoStore(t *testing.T) {
	testMultiRepoStore(t, func() MultiRepoStoreImporter {
		tmpDir, err := ioutil.TempDir("", "srclib-TestFlatFileMultiRepoStore")
		if err != nil {
			t.Fatal(err)
		}
		fs := rwvfs.OS(tmpDir)
		setCreateParentDirs(fs)
		return NewFlatFileMultiRepoStore(rwvfs.Sub(fs, "multirepo"), nil)
	})
}
