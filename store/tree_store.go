package store

import (
	"sourcegraph.com/sourcegraph/srclib/graph"
	"sourcegraph.com/sourcegraph/srclib/unit"
)

// A TreeStore stores and accesses srclib build data for an arbitrary
// source tree (consisting of any number of source units).
type TreeStore interface {
	// Unit gets a single unit by its unit type and name. If no such
	// unit exists, an error satisfying IsNotExist is returned.
	Unit(unit.Key) (*unit.SourceUnit, error)

	// Units returns all units that match the filter.
	Units(...UnitFilter) ([]*unit.SourceUnit, error)

	// UnitStore's methods call the corresponding methods on the
	// UnitStore of each source unit contained within this tree. The
	// combined results are returned (in undefined order).
	UnitStore
}

// A TreeImporter imports srclib build data for a source unit into a
// TreeStore.
type TreeImporter interface {
	// Import imports a source unit and its graph data into the
	// store. If Import is called with a nil SourceUnit and output
	// data, the importer considers the tree to have no source units
	// until others are imported in the future (this makes it possible
	// to distinguish between a tree that has no source units and a
	// tree whose source units simply haven't been imported yet).
	Import(*unit.SourceUnit, graph.Output) error
}

// A TreeStoreImporter implements both TreeStore and TreeImporter.
type TreeStoreImporter interface {
	TreeStore
	TreeImporter
}

// A treeStores is a TreeStore whose methods call the
// corresponding method on each of the tree stores returned by the
// treeStores func.
type treeStores struct {
	opener treeStoreOpener
}

var _ TreeStore = (*treeStores)(nil)

func (s treeStores) Unit(key unit.Key) (*unit.SourceUnit, error) {
	tss, err := openTreeStores(s.opener, ByUnits(key.ID2()))
	if err != nil {
		if isStoreNotExist(err) {
			return nil, errUnitNotExist
		}
		return nil, err
	}

	for commitID, ts := range tss {
		if key.CommitID != commitID {
			continue
		}
		unit, err := ts.Unit(key)
		if err != nil {
			if IsNotExist(err) || isStoreNotExist(err) {
				continue
			}
			return nil, err
		}
		if unit.CommitID == "" {
			unit.CommitID = commitID
		}
		return unit, nil
	}
	return nil, errUnitNotExist
}

func (s treeStores) Units(f ...UnitFilter) ([]*unit.SourceUnit, error) {
	tss, err := openTreeStores(s.opener, f)
	if err != nil {
		return nil, err
	}

	var allUnits []*unit.SourceUnit
	for commitID, ts := range tss {
		units, err := ts.Units(f...)
		if err != nil {
			return nil, err
		}
		for _, unit := range units {
			unit.CommitID = commitID
		}
		allUnits = append(allUnits, units...)
	}
	return allUnits, nil
}

func (s treeStores) Def(key graph.DefKey) (*graph.Def, error) {
	if err := checkDefKeyValidForRepoStore(key); err != nil {
		return nil, err
	}

	tss, err := openTreeStores(s.opener, []interface{}{ByCommitID(key.CommitID), ByUnits(unit.ID2{Type: key.UnitType, Name: key.Unit})})
	if err != nil {
		if isStoreNotExist(err) {
			return nil, errDefNotExist
		}
		return nil, err
	}

	for commitID, ts := range tss {
		if key.CommitID != commitID {
			continue
		}
		def, err := ts.Def(key)
		if err != nil {
			if IsNotExist(err) || isStoreNotExist(err) {
				continue
			}
			return nil, err
		}
		def.CommitID = commitID
		return def, nil
	}
	return nil, errDefNotExist
}

func (s treeStores) Defs(f ...DefFilter) ([]*graph.Def, error) {
	tss, err := openTreeStores(s.opener, f)
	if err != nil {
		return nil, err
	}

	var allDefs []*graph.Def
	for commitID, ts := range tss {
		defs, err := ts.Defs(f...)
		if err != nil {
			return nil, err
		}
		for _, def := range defs {
			def.CommitID = commitID
		}
		allDefs = append(allDefs, defs...)
	}
	return allDefs, nil
}

func (s treeStores) Refs(f ...RefFilter) ([]*graph.Ref, error) {
	tss, err := openTreeStores(s.opener, f)
	if err != nil {
		return nil, err
	}

	var allRefs []*graph.Ref
	for commitID, ts := range tss {
		setImpliedCommitID(f, commitID)
		refs, err := ts.Refs(f...)
		if err != nil {
			return nil, err
		}
		for _, ref := range refs {
			ref.CommitID = commitID
		}
		allRefs = append(allRefs, refs...)
	}
	return allRefs, nil
}

// checkDefKeyValidForTreeStore returns an *InvalidKeyError if the def
// key is underspecified for use in (TreeStore).Def.
func checkDefKeyValidForTreeStore(key graph.DefKey) error {
	if err := checkDefKeyValidForUnitStore(key); err != nil {
		return err
	}
	if key.Unit == "" {
		return &InvalidKeyError{"empty DefKey.Unit"}
	}
	if key.UnitType == "" {
		return &InvalidKeyError{"empty DefKey.UnitType"}
	}
	return nil
}
