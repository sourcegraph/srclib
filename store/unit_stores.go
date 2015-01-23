package store

import "sourcegraph.com/sourcegraph/srclib/unit"

// scopeUnits returns a list of units that are matched by the
// filters. If potentially all units could match, or if enough units
// could potentially match that it would probably be cheaper to
// iterate through all of them, then a nil slice is returned. If none
// match, an empty slice is returned.
//
// TODO(sqs): return an error if the filters are mutually exclusive?
func scopeUnits(filters []interface{}) ([]unit.ID2, error) {
	unitIDs := map[unit.ID2]struct{}{}

	for _, f := range filters {
		switch f := f.(type) {
		case ByUnitFilter:
			u := unit.ID2{Type: f.ByUnitType(), Name: f.ByUnit()}
			if len(unitIDs) == 0 {
				unitIDs[u] = struct{}{}
			} else if _, dup := unitIDs[u]; !dup {
				// Mutually exclusive unit IDs.
				return []unit.ID2{}, nil
			}
		}
	}

	if len(unitIDs) == 0 {
		// Scope includes potentially all units.
		return nil, nil
	}

	ids := make([]unit.ID2, 0, len(unitIDs))
	for u := range unitIDs {
		ids = append(ids, u)
	}
	return ids, nil
}

// A unitStoreOpener opens the UnitStore for the specified source
// unit.
type unitStoreOpener interface {
	openUnitStore(unit.ID2) (UnitStore, error)
	openAllUnitStores() (map[unit.ID2]UnitStore, error)
}

// openUnitStores is a helper func that calls o.openUnitStore for each
// unit returned by scopeUnits(filters...).
func openUnitStores(o unitStoreOpener, filters interface{}) (map[unit.ID2]UnitStore, error) {
	unitIDs, err := scopeUnits(storeFilters(filters))
	if err != nil {
		return nil, err
	}

	if unitIDs == nil {
		return o.openAllUnitStores()
	}

	uss := make(map[unit.ID2]UnitStore, len(unitIDs))
	for _, u := range unitIDs {
		var err error
		uss[u], err = o.openUnitStore(u)
		if err != nil {
			return nil, err
		}
	}
	return uss, nil
}
