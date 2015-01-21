package store

import "sourcegraph.com/sourcegraph/srclib/unit"

type MockTreeStore struct {
	Unit_  func(unit.Key) (*unit.SourceUnit, error)
	Units_ func(...UnitFilter) ([]*unit.SourceUnit, error)
	MockUnitStore
}

func (m MockTreeStore) Unit(key unit.Key) (*unit.SourceUnit, error) {
	return m.Unit_(key)
}

func (m MockTreeStore) Units(f ...UnitFilter) ([]*unit.SourceUnit, error) {
	return m.Units_(f...)
}

var _ TreeStore = MockTreeStore{}
