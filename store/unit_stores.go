package store

type unitID struct{ unitType, unit string }

// scopeUnits returns a list of units that are matched by the
// filters. If potentially all units could match, or if enough units
// could potentially match that it would probably be cheaper to
// iterate through all of them, then a nil slice is returned. If none
// match, an empty slice is returned.
//
// TODO(sqs): return an error if the filters are mutually exclusive?
func scopeUnits(filters ...storesFilter) ([]unitID, error) {
	unitIDs := map[unitID]struct{}{}

	for _, f := range filters {
		switch f := f.(type) {
		case ByUnitFilter:
			u := unitID{unitType: f.ByUnitType(), unit: f.ByUnit()}
			if len(unitIDs) == 0 {
				unitIDs[u] = struct{}{}
			} else if _, dup := unitIDs[u]; !dup {
				// Mutually exclusive unit IDs.
				return []unitID{}, nil
			}
		}
	}

	if len(unitIDs) == 0 {
		// Scope includes potentially all units.
		return nil, nil
	}

	ids := make([]unitID, 0, len(unitIDs))
	for unitID := range unitIDs {
		ids = append(ids, unitID)
	}
	return ids, nil
}

// A unitStoreOpener opens the UnitStore for the specified source
// unit.
type unitStoreOpener interface {
	openUnitStore(unitID) (UnitStore, error)
	openAllUnitStores() (map[unitID]UnitStore, error)
}

// openUnitStores is a helper func that calls o.openUnitStore for each
// unit returned by scopeUnits(filters...).
func openUnitStores(o unitStoreOpener, filters ...storesFilter) (map[unitID]UnitStore, error) {
	unitIDs, err := scopeUnits(filters...)
	if err != nil {
		return nil, err
	}

	if unitIDs == nil {
		return o.openAllUnitStores()
	}

	uss := make(map[unitID]UnitStore, len(unitIDs))
	for _, u := range unitIDs {
		var err error
		uss[u], err = o.openUnitStore(u)
		if err != nil {
			return nil, err
		}
	}
	return uss, nil
}
