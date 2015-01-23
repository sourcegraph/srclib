package store

import "testing"

func TestFlatFileUnitStore(t *testing.T) {
	useIndexedStore = false
	testUnitStore(t, func() unitStoreImporter {
		return &flatFileUnitStore{fs: newTestFS()}
	})
}

func TestFlatFileTreeStore(t *testing.T) {
	useIndexedStore = false
	testTreeStore(t, func() treeStoreImporter {
		return newFlatFileTreeStore(newTestFS())
	})
}

func TestFlatFileRepoStore(t *testing.T) {
	useIndexedStore = false
	testRepoStore(t, func() RepoStoreImporter {
		return NewFlatFileRepoStore(newTestFS())
	})
}

func TestFlatFileMultiRepoStore(t *testing.T) {
	useIndexedStore = false
	testMultiRepoStore(t, func() MultiRepoStoreImporter {
		return NewFlatFileMultiRepoStore(newTestFS())
	})
}
