package store

import "testing"

func TestIndexedUnitStore(t *testing.T) {
	useIndexedStore = true
	testUnitStore(t, func() unitStoreImporter {
		return newIndexedUnitStore(newTestFS())
	})
}

func TestIndexedTreeStore(t *testing.T) {
	useIndexedStore = true
	testTreeStore(t, func() treeStoreImporter {
		return newIndexedTreeStore(newTestFS())
	})
}

func TestIndexedFlatFileTreeStore(t *testing.T) {
	useIndexedStore = true
	testTreeStore(t, func() treeStoreImporter {
		return newFlatFileTreeStore(newTestFS())
	})
}

func TestIndexedFlatFileRepoStore(t *testing.T) {
	useIndexedStore = true
	testRepoStore(t, func() RepoStoreImporter {
		return NewFlatFileRepoStore(newTestFS())
	})
}

func TestIndexedFlatFileMultiRepoStore(t *testing.T) {
	useIndexedStore = true
	testMultiRepoStore(t, func() MultiRepoStoreImporter {
		return NewFlatFileMultiRepoStore(newTestFS())
	})
}
