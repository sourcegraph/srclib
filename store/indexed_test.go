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

func TestIndexedFSTreeStore(t *testing.T) {
	useIndexedStore = true
	testTreeStore(t, func() treeStoreImporter {
		return newFSTreeStore(newTestFS())
	})
}

func TestIndexedFSRepoStore(t *testing.T) {
	useIndexedStore = true
	testRepoStore(t, func() RepoStoreImporter {
		return NewFSRepoStore(newTestFS())
	})
}

func TestIndexedFSMultiRepoStore(t *testing.T) {
	useIndexedStore = true
	testMultiRepoStore(t, func() MultiRepoStoreImporter {
		return NewFSMultiRepoStore(newTestFS())
	})
}
