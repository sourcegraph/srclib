package store

import (
	"errors"
	"fmt"
	"log"
	"runtime"

	"code.google.com/p/rog-go/parallel"

	"sort"
	"strings"

	"sourcegraph.com/sourcegraph/rwvfs"
	"sourcegraph.com/sourcegraph/srclib/graph"
	"sourcegraph.com/sourcegraph/srclib/unit"
)

// An indexedTreeStore is a VFS-backed tree store that generates
// indexes to provide efficient lookups.
//
// It wraps a fsTreeStore and intercepts calls to Def, Defs,
// Refs, etc., using its indexes to satisfy those queries efficiently
// where possible. Otherwise it invokes the underlying store to
// perform a full scan (over all source units in the tree).
type indexedTreeStore struct {
	// indexes is all of the indexes that should be built, written,
	// and read from. It contains all indexes for all types of data
	// (e.g., def indexes, ref indexes, etc.).
	indexes []interface{}

	*fsTreeStore
}

var _ TreeStore = (*indexedTreeStore)(nil)

// newIndexedTreeStore creates a new indexed tree store that stores
// data and indexes in fs.
func newIndexedTreeStore(fs rwvfs.FileSystem) TreeStoreImporter {
	return &indexedTreeStore{
		indexes: []interface{}{
			&unitFilesIndex{},
		},
		fsTreeStore: newFSTreeStore(fs),
	}
}

// errNotIndexed occurs when that a query was unable to be performed
// using an index. In most cases, it indicates that the caller should
// perform the query using a full scan of all of the data.
var errNotIndexed = errors.New("no index satisfies query")

// unitIDs returns the source unit IDs that satisfy the unit
// filters. If possible, it uses indexes instead of performing a full
// scan of the source unit files.
//
// If indexOnly is specified, only the index will be consulted. If a
// full scan would otherwise occur, errNotIndexed is returned.
func (s *indexedTreeStore) unitIDs(indexOnly bool, fs ...UnitFilter) ([]unit.ID2, error) {
	vlog.Printf("indexedTreeStore.unitIDs(indexOnly=%v, %v)", indexOnly, fs)

	// Try to find an index that covers this query.
	if dx := bestCoverageUnitIndex(s.indexes, fs); dx != nil {
		if px, ok := dx.(persistedIndex); ok && !px.Ready() {
			if err := readIndex(s.fs, px); err != nil {
				return nil, err
			}
		}
		vlog.Printf("indexedTreeStore.unitIDs(%v): Found covering index %v.", fs, dx)
		return dx.Units(fs...)
	}
	if indexOnly {
		return nil, errNotIndexed
	}

	// Fall back to full scan.
	vlog.Printf("indexedTreeStore.unitIDs(%v): No covering indexes found; performing full scan.", fs)
	var unitIDs []unit.ID2
	units, err := s.fsTreeStore.Units(fs...)
	if err != nil {
		return nil, err
	}
	for _, u := range units {
		unitIDs = append(unitIDs, u.ID2())
	}
	return unitIDs, nil
}

func (s *indexedTreeStore) Units(fs ...UnitFilter) ([]*unit.SourceUnit, error) {
	// Attempt to use the index.
	scopeUnits, err := s.unitIDs(false, fs...)
	if err != nil {
		if err == errNotIndexed {
			return s.fsTreeStore.Units(fs...)
		}
		return nil, err
	}

	fs = append(fs, ByUnits(scopeUnits...))
	return s.fsTreeStore.Units(fs...)
}

func (s *indexedTreeStore) Defs(fs ...DefFilter) ([]*graph.Def, error) {
	vlog.Printf("indexedTreeStore.Defs(%v)", fs)

	// We have File->Unit index (that tells us which source units
	// include a given file). If there's a ByFiles DefFilter, then we
	// can convert that filter into a ByUnits scope filter (which is
	// more efficient) by consulting the File->Unit index.

	var ufs []UnitFilter
	for _, f := range fs {
		if uf, ok := f.(UnitFilter); ok {
			ufs = append(ufs, uf)
		}
	}

	// No indexes found that we can exploit here; forward to the
	// underlying store.
	if len(ufs) == 0 {
		vlog.Printf("indexedTreeStore.Defs(%v): No unit indexes found to narrow scope; forwarding to underlying store.", fs)
		return s.fsTreeStore.Defs(fs...)
	}

	// Find which source units match the unit filters; we'll restrict
	// our defs query to those source units.
	scopeUnits, err := s.unitIDs(false, ufs...)
	if err != nil {
		return nil, err
	}

	// Add ByUnits filters that were implied by ByFiles (and other
	// UnitFilters).
	//
	// If scopeUnits is empty, the empty ByUnits filter will result in
	// the query matching nothing, which is the desired behavior.
	vlog.Printf("indexedTreeStore.Defs(%v): Adding equivalent ByUnits filters to scope to units %+v.", fs, scopeUnits)
	fs = append(fs, ByUnits(scopeUnits...))

	// Pass the now more narrowly scoped query onto the underlying store.
	return s.fsTreeStore.Defs(fs...)
}

func (s *indexedTreeStore) Refs(fs ...RefFilter) ([]*graph.Ref, error) {
	// We have File->Unit index (that tells us which source units
	// include a given file). If there's a ByFiles RefFilter, then we
	// can convert that filter into a ByUnits scope filter (which is
	// more efficient) by consulting the File->Unit index.

	var ufs []UnitFilter
	for _, f := range fs {
		if uf, ok := f.(UnitFilter); ok {
			ufs = append(ufs, uf)
		}
	}

	// No indexes found that we can exploit here; forward to the
	// underlying store.
	if len(ufs) == 0 {
		return s.fsTreeStore.Refs(fs...)
	}

	// Find which source units match the unit filters; we'll restrict
	// our refs query to those source units.
	scopeUnits, err := s.unitIDs(false, ufs...)
	if err != nil {
		return nil, err
	}

	// Add ByUnits filters that were implied by ByFiles (and other
	// UnitFilters).
	//
	// If scopeUnits is empty, the empty ByUnits filter will result in
	// the query matching nothing, which is the desired behavior.
	fs = append(fs, ByUnits(scopeUnits...))

	// Pass the now more narrowly scoped query onto the underlying store.
	return s.fsTreeStore.Refs(fs...)
}

func (s *indexedTreeStore) Import(u *unit.SourceUnit, data graph.Output) error {
	if err := s.fsTreeStore.Import(u, data); err != nil {
		return err
	}

	s.checkSourceUnitFiles(u, data)

	if err := s.writeUnitIndexes(u); err != nil {
		return err
	}

	return nil
}

// checkSourceUnitFiles warns if any files appear in graph data but
// are not in u.Files.
func (s *indexedTreeStore) checkSourceUnitFiles(u *unit.SourceUnit, data graph.Output) {
	if u == nil {
		return
	}

	graphFiles := make(map[string]struct{}, len(u.Files))
	for _, def := range data.Defs {
		graphFiles[def.File] = struct{}{}
	}
	for _, ref := range data.Refs {
		graphFiles[ref.File] = struct{}{}
	}
	for _, doc := range data.Docs {
		graphFiles[doc.File] = struct{}{}
	}
	for _, ann := range data.Anns {
		graphFiles[ann.File] = struct{}{}
	}
	delete(graphFiles, "")

	unitFiles := make(map[string]struct{}, len(u.Files))
	for _, f := range u.Files {
		unitFiles[f] = struct{}{}
	}
	if u.Dir != "" {
		unitFiles[u.Dir] = struct{}{}
	}

	var missingFiles []string
	for f := range graphFiles {
		if _, present := unitFiles[f]; !present {
			missingFiles = append(missingFiles, f)
		}
	}
	if len(missingFiles) > 0 {
		sort.Strings(missingFiles)
		log.Printf("Warning: The graph output (defs/refs/docs/anns) for source unit %+v contain %d references to files that are not present in the source unit's Files list. Indexed lookups by any of these missing files will return no results. To fix this, ensure that the source unit's Files list includes all files that appear in the graph output. The missing files are: %s.", u.ID2(), len(missingFiles), strings.Join(missingFiles, " "))
	}
}

// writeUnitIndexes builds every unit index in s.indexes and writes
// them to their backing files. Because imports are performed on a
// per-unit basis, it rebuilds the index from scratch after each
// source unit is finished importing.
func (s *indexedTreeStore) writeUnitIndexes(u *unit.SourceUnit) error {
	// TODO(sqs): there's a race condition here if multiple imports
	// are running concurrently, they could clobber each other's
	// indexes. (S3 is eventually consistent.)
	units, err := s.fsTreeStore.Units()
	if err != nil {
		return err
	}

	vlog.Printf("indexedTreeStore.writeUnitIndexes(%v): building indexes for all units: %v", u, units)

	par := parallel.NewRun(runtime.GOMAXPROCS(0))
	for _, x := range s.indexes {
		if x, ok := x.(unitIndex); ok {
			if bx, ok := x.(unitIndexBuilder); ok {
				par.Do(func() error {
					if err := bx.Build(units); err != nil {
						return err
					}
					if px, ok := x.(persistedIndex); ok {
						if err := writeIndex(s.fs, px); err != nil {
							return err
						}
					}
					return nil
				})
			}
		}
	}
	return par.Wait()
}

//func defsAtUnitOffsets(o unitStoreOpener

// An indexedUnitStore is a VFS-backed unit store that generates
// indexes to provide efficient lookups.
//
// It wraps a fsUnitStore and intercepts calls to Def, Defs,
// Refs, etc., using its indexes to satisfy those queries efficiently
// where possible. Otherwise it invokes the underlying store to
// perform a full scan.
type indexedUnitStore struct {
	// indexes is all of the indexes that should be built, written,
	// and read from. It contains all indexes for all types of data
	// (e.g., def indexes, ref indexes, etc.).
	indexes []interface{}

	*fsUnitStore
}

var _ UnitStore = (*indexedUnitStore)(nil)

// newIndexedUnitStore creates a new indexed unit store that stores
// data and indexes in fs.
func newIndexedUnitStore(fs rwvfs.FileSystem) UnitStoreImporter {
	return &indexedUnitStore{
		indexes: []interface{}{
			&defPathIndex{},
			&refFileIndex{},
			&defFilesIndex{
				filters: []DefFilter{
					DefFilterFunc(func(def *graph.Def) bool {
						return def.Exported || !def.Local
					}),
				},
				perFile: 7,
			},
		},
		fsUnitStore: &fsUnitStore{fs: fs},
	}
}

const indexFilename = "%s.idx"

// Def implements UnitStore.
func (s *indexedUnitStore) Def(key graph.DefKey) (def *graph.Def, err error) {
	defs, err := s.Defs(ByDefPath(key.Path))
	if err != nil {
		return nil, err
	}
	if len(defs) == 0 {
		return nil, errDefNotExist
	}
	return defs[0], nil
}

func (s *indexedUnitStore) Defs(fs ...DefFilter) ([]*graph.Def, error) {
	// Try to find an index that covers this query.
	if dx := bestCoverageDefIndex(s.indexes, fs); dx != nil {
		if px, ok := dx.(persistedIndex); ok && !px.Ready() {
			if err := readIndex(s.fs, px); err != nil {
				return nil, err
			}
		}
		ofs, err := dx.Defs(fs...)
		if err != nil {
			return nil, err
		}
		return s.defsAtOffsets(ofs, fs)
	}

	// Fall back to full scan.
	return s.fsUnitStore.Defs(fs...)
}

// Refs implements UnitStore.
func (s *indexedUnitStore) Refs(fs ...RefFilter) ([]*graph.Ref, error) {
	// Try to find an index that covers this query.
	if dx := bestCoverageRefIndex(s.indexes, fs); dx != nil {
		if px, ok := dx.(persistedIndex); ok && !px.Ready() {
			if err := readIndex(s.fs, px); err != nil {
				return nil, err
			}
		}
		brs, err := dx.Refs(fs...)
		if err != nil {
			return nil, err
		}
		return s.refsAtByteRanges(brs, fs)
	}

	// Fall back to full scan.
	return s.fsUnitStore.Refs(fs...)
}

// Import calls to the underlying fsUnitStore to write the def
// and ref data files. It also builds and writes the indexes.
func (s *indexedUnitStore) Import(data graph.Output) error {
	cleanForImport(&data, "", "", "")

	// TODO(sqs): parallelize

	defOfs, err := s.fsUnitStore.writeDefs(&data)
	if err != nil {
		return err
	}
	if err := s.writeDefIndexes(&data, defOfs); err != nil {
		return err
	}

	fbr, err := s.fsUnitStore.writeRefs(&data)
	if err != nil {
		return err
	}
	if err := s.writeRefIndexes(&data, fbr); err != nil {
		return err
	}

	return nil
}

// writeDefIndexes builds every def index in s.indexes and writes them
// to their backing files.
func (s *indexedUnitStore) writeDefIndexes(data *graph.Output, ofs byteOffsets) error {
	par := parallel.NewRun(runtime.GOMAXPROCS(0))
	for _, x := range s.indexes {
		if x, ok := x.(defIndex); ok {
			if bx, ok := x.(graphIndexBuilder); ok {
				par.Do(func() error {
					if err := bx.Build(data, ofs); err != nil {
						return err
					}
					if px, ok := x.(persistedIndex); ok {
						if err := writeIndex(s.fs, px); err != nil {
							return err
						}
					}
					return nil
				})
			}
		}
	}
	return par.Wait()
}

// writeRefIndexes builds every ref index in s.indexes and writes them
// to their backing files.
func (s *indexedUnitStore) writeRefIndexes(data *graph.Output, fbr fileByteRanges) error {
	if len(s.indexes) == 0 {
		return nil
	}
	par := parallel.NewRun(runtime.GOMAXPROCS(0))
	for _, x := range s.indexes {
		if x, ok := x.(refIndex); ok {
			par.Do(func() error {
				if err := x.Build(data, fbr); err != nil {
					return err
				}
				if px, ok := x.(persistedIndex); ok {
					if err := writeIndex(s.fs, px); err != nil {
						return err
					}
				}
				return nil
			})
		}
	}
	return par.Wait()
}

// writeIndex calls x.Write with the index's backing file.
func writeIndex(fs rwvfs.FileSystem, x persistedIndex) (err error) {
	f, err := fs.Create(fmt.Sprintf(indexFilename, x.Name()))
	if err != nil {
		return err
	}
	defer func() {
		err2 := f.Close()
		if err == nil {
			err = err2
		}
	}()
	return x.Write(f)
}

// readIndex calls x.Read with the index's backing file.
func readIndex(fs rwvfs.FileSystem, x persistedIndex) (err error) {
	if x.Ready() {
		panic("x is already Ready; attempted to read it again")
	}

	f, err := fs.Open(fmt.Sprintf(indexFilename, x.Name()))
	if err != nil {
		return err
	}
	defer func() {
		err2 := f.Close()
		if err == nil {
			err = err2
		}
	}()

	return x.Read(f)
}

func (s *indexedUnitStore) String() string { return "indexedUnitStore" }
