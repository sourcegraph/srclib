package store

import (
	"io/ioutil"
	"testing"

	"sourcegraph.com/sourcegraph/rwvfs"
)

func TestFlatFileUnitStore(t *testing.T) {
	testUnitStore(t, func() UnitStore {
		tmpDir, err := ioutil.TempDir("", "srclib-TestFlatFileUnitStore")
		if err != nil {
			t.Fatal(err)
		}
		return &flatFileUnitStore{rwvfs.OS(tmpDir)}
	})
}
