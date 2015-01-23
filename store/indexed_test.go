package store

import (
	"testing"

	"sourcegraph.com/sourcegraph/rwvfs"
)

func TestIndexedUnitStore(t *testing.T) {
	useIndexedUnitStore = true
	testUnitStore(t, func() unitStoreImporter {
		return newIndexedUnitStore(rwvfs.Map(map[string]string{}), JSONCodec{})
	})
}

func TestIndexedFlatFileTreeStore(t *testing.T) {
	useIndexedUnitStore = true
	testTreeStore(t, func() treeStoreImporter {
		return newFlatFileTreeStore(newTestFS(), nil)
	})
}

func TestIndexedFlatFileRepoStore(t *testing.T) {
	useIndexedUnitStore = true
	testRepoStore(t, func() RepoStoreImporter {
		return NewFlatFileRepoStore(newTestFS(), nil)
	})
}

func TestIndexedFlatFileMultiRepoStore(t *testing.T) {
	useIndexedUnitStore = true
	testMultiRepoStore(t, func() MultiRepoStoreImporter {
		return NewFlatFileMultiRepoStore(newTestFS(), nil)
	})
}
