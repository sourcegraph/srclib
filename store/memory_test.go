package store

import "testing"

func TestMemoryUnitStore(t *testing.T) {
	testUnitStore(t, func() UnitStore {
		return &memoryUnitStore{}
	})
}
