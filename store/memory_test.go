package store

import "testing"

func TestMemoryUnitStore(t *testing.T) {
	testUnitStore(t, func() unitStoreImporter {
		return &memoryUnitStore{}
	})
}

func TestMemoryTreeStore(t *testing.T) {
	testTreeStore(t, func() treeStoreImporter {
		return newMemoryTreeStore()
	})
}

func TestMemoryRepoStore(t *testing.T) {
	testRepoStore(t, func() repoStoreImporter {
		return newMemoryRepoStore()
	})
}
