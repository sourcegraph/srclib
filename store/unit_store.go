package store

import (
	"sync"

	"code.google.com/p/rog-go/parallel"
	"sourcegraph.com/sourcegraph/srclib/graph"
)

// A UnitStore stores and accesses srclib build data for a single
// source unit.
type UnitStore interface {
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

func (s unitStores) Defs(fs ...DefFilter) ([]*graph.Def, error) {
	uss, err := openUnitStores(s.opener, fs)
	if err != nil {
		return nil, err
	}

	var (
		allDefs   []*graph.Def
		allDefsMu sync.Mutex
	)
	par := parallel.NewRun(storeFetchPar)
	for u_, us_ := range uss {
		u, us := u_, us_
		if us == nil {
			continue
		}

		par.Do(func() error {
			// If the filters list includes any unitDefOffsetsFilter, then
			// clone and transform that filter into a defOffsetsFilter
			// that only includes offsets to fetch from the source unit of
			// the store. (This is necessary because the store doesn't
			// know which unit it holds data for, so it doesn't know which
			// offsets to look up given just a unitDefOffsetsFilter
			// map[unit.ID2]byteOffsets map.)
			fs2 := fs
			for i, f := range fs {
				switch f := f.(type) {
				case unitDefOffsetsFilter:
					fs2 = make([]DefFilter, len(fs))
					for j := range fs {
						if j == i {
							// Transform unitDefOffsetsFilter to
							// defOffsetsFilter for the current source
							// unit.
							fs2[j] = defOffsetsFilter(f[u])
						} else {
							// Copy existing filter if it's of any other
							// type. (Assumes the filters list contains no
							// more than 1 unitDefOffsetsFilters.)
							fs2[j] = fs[j]
						}
					}
				}
			}

			defs, err := us.Defs(fs2...)
			if err != nil && !isStoreNotExist(err) {
				return err
			}
			for _, def := range defs {
				def.UnitType = u.Type
				def.Unit = u.Name
			}
			allDefsMu.Lock()
			allDefs = append(allDefs, defs...)
			allDefsMu.Unlock()
			return nil
		})
	}
	err = par.Wait()
	return allDefs, err
}

var c_unitStores_Refs_last_numUnitsQueried = 0

func (s unitStores) Refs(f ...RefFilter) ([]*graph.Ref, error) {
	uss, err := openUnitStores(s.opener, f)
	if err != nil {
		return nil, err
	}

	c_unitStores_Refs_last_numUnitsQueried = 0
	var allRefs []*graph.Ref
	for u, us := range uss {
		if us == nil {
			continue
		}

		c_unitStores_Refs_last_numUnitsQueried++
		setImpliedUnit(f, u)
		refs, err := us.Refs(f...)
		if err != nil && !isStoreNotExist(err) {
			return nil, err
		}
		for _, ref := range refs {
			ref.UnitType = u.Type
			ref.Unit = u.Name
			if ref.DefUnitType == "" {
				ref.DefUnitType = u.Type
			}
			if ref.DefUnit == "" {
				ref.DefUnit = u.Name
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
