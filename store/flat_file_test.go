package store

import "testing"

func TestFlatFileUnitStore(t *testing.T) {
	useIndexedUnitStore = false
	testUnitStore(t, func() unitStoreImporter {
		return &flatFileUnitStore{fs: newTestFS(), codec: JSONCodec{}}
	})
}

func TestFlatFileTreeStore(t *testing.T) {
	useIndexedUnitStore = false
	testTreeStore(t, func() treeStoreImporter {
		return newFlatFileTreeStore(newTestFS(), nil)
	})
}

func TestFlatFileRepoStore(t *testing.T) {
	useIndexedUnitStore = false
	testRepoStore(t, func() RepoStoreImporter {
		return NewFlatFileRepoStore(newTestFS(), nil)
	})
}

func TestFlatFileMultiRepoStore(t *testing.T) {
	useIndexedUnitStore = false
	testMultiRepoStore(t, func() MultiRepoStoreImporter {
		return NewFlatFileMultiRepoStore(newTestFS(), nil)
	})
}
