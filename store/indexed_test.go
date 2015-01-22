package store

import (
	"testing"

	"sourcegraph.com/sourcegraph/rwvfs"
)

func TestIndexedUnitStore(t *testing.T) {
	testUnitStore(t, func() unitStoreImporter {
		return newIndexedUnitStore(rwvfs.Map(map[string]string{}), JSONCodec{})
	})
}
