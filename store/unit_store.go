package store

import "sourcegraph.com/sourcegraph/srclib/graph"

// A UnitStore stores and accesses srclib build data for a single
// source unit.
type UnitStore interface {
	// Def gets a single def by its key. If no such def exists, an
	// error satisfying IsNotExist is returned.
	Def(graph.DefKey) (*graph.Def, error)

	// Defs returns all defs that match the filter.
	Defs(...DefFilter) ([]*graph.Def, error)

	// Refs returns all refs that match the filter.
	Refs(...RefFilter) ([]*graph.Ref, error)

	// TODO(sqs): how to deal with depresolve and other non-graph
	// data?
}

// A UnitImporter imports srclib build data for a single source unit
// into a UnitStore.
type UnitImporter interface {
	// Import imports defs, refs, etc., into the store. It overwrites
	// all existing data for this source unit (and at the commit, if
	// applicable).
	Import(graph.Output) error
}

// A UnitStoreImporter implements both UnitStore and UnitImporter.
type UnitStoreImporter interface {
	UnitStore
	UnitImporter
}

// A unitStores is a UnitStore whose methods call the
// corresponding method on each of the unit stores returned by the
// unitStores func.
type unitStores struct {
	opener unitStoreOpener
}

var _ UnitStore = (*unitStores)(nil)

func (s unitStores) Def(key graph.DefKey) (*graph.Def, error) {
	if err := checkDefKeyValidForTreeStore(key); err != nil {
		return nil, err
	}

	uss, err := openUnitStores(s.opener, ByUnit(key.UnitType, key.Unit))
	if err != nil {
		if isStoreNotExist(err) {
			return nil, errDefNotExist
		}
		return nil, err
	}

	for u, us := range uss {
		if key.UnitType != u.unitType || key.Unit != u.unit {
			continue
		}
		def, err := us.Def(key)
		if err != nil {
			if IsNotExist(err) || isStoreNotExist(err) {
				continue
			}
			return nil, err
		}
		def.UnitType = u.unitType
		def.Unit = u.unit
		return def, nil
	}
	return nil, errDefNotExist
}

func (s unitStores) Defs(f ...DefFilter) ([]*graph.Def, error) {
	uss, err := openUnitStores(s.opener, f)
	if err != nil {
		return nil, err
	}

	var allDefs []*graph.Def
	for u, us := range uss {
		defs, err := us.Defs(f...)
		if err != nil {
			return nil, err
		}
		for _, def := range defs {
			def.UnitType = u.unitType
			def.Unit = u.unit
		}
		allDefs = append(allDefs, defs...)
	}
	return allDefs, nil
}

func (s unitStores) Refs(f ...RefFilter) ([]*graph.Ref, error) {
	uss, err := openUnitStores(s.opener, f)
	if err != nil {
		return nil, err
	}

	var allRefs []*graph.Ref
	for u, us := range uss {
		setImpliedUnit(f, u)
		refs, err := us.Refs(f...)
		if err != nil {
			return nil, err
		}
		for _, ref := range refs {
			ref.UnitType = u.unitType
			ref.Unit = u.unit
			if ref.DefUnitType == "" {
				ref.DefUnitType = u.unitType
			}
			if ref.DefUnit == "" {
				ref.DefUnit = u.unit
			}
		}
		allRefs = append(allRefs, refs...)
	}
	return allRefs, nil
}

func cleanForImport(data *graph.Output, repo, unitType, unit string) {
	for _, def := range data.Defs {
		def.Unit = ""
		def.UnitType = ""
		def.Repo = ""
		def.CommitID = ""
	}
	for _, ref := range data.Refs {
		ref.Unit = ""
		ref.UnitType = ""
		ref.Repo = ""
		ref.CommitID = ""
		if repo != "" && ref.DefRepo == repo {
			ref.DefRepo = ""
		}
		if unitType != "" && ref.DefUnitType == unitType {
			ref.DefUnitType = ""
		}
		if unit != "" && ref.DefUnit == unit {
			ref.DefUnit = ""
		}
	}
	for _, doc := range data.Docs {
		doc.Unit = ""
		doc.UnitType = ""
		doc.Repo = ""
		doc.CommitID = ""
	}
	for _, ann := range data.Anns {
		ann.Unit = ""
		ann.UnitType = ""
		ann.Repo = ""
		ann.CommitID = ""
	}
}

// checkDefKeyValidForTreeStore returns an *InvalidKeyError if the def
// key is underspecified for use in (UnitStore).Def.
func checkDefKeyValidForUnitStore(key graph.DefKey) error {
	if key.Path == "" {
		return &InvalidKeyError{"empty DefKey.Path"}
	}
	return nil
}
