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
	everHadAny := false // whether unitIDs ever contained any units

	for _, f := range filters {
		switch f := f.(type) {
		case ByUnitsFilter:
			if len(unitIDs) == 0 && !everHadAny {
				everHadAny = true
				for _, u := range f.ByUnits() {
					unitIDs[u] = struct{}{}
				}
			} else {
				// Intersect.
				newUnitIDs := make(map[unit.ID2]struct{}, (len(unitIDs)+len(f.ByUnits()))/2)
				for _, u := range f.ByUnits() {
					if _, present := unitIDs[u]; present {
						newUnitIDs[u] = struct{}{}
					}
				}
				unitIDs = newUnitIDs
			}
		}
	}

	if len(unitIDs) == 0 && !everHadAny {
		// No unit scoping filters were present, so scope includes
		// potentially all units.
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
	openUnitStore(unit.ID2) UnitStore
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
		uss[u] = o.openUnitStore(u)
	}
	return uss, nil
}
