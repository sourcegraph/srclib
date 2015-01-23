package store

import (
	"fmt"
	"runtime"

	"code.google.com/p/rog-go/parallel"

	"sourcegraph.com/sourcegraph/rwvfs"
	"sourcegraph.com/sourcegraph/srclib/graph"
)

// An indexedUnitStore is a VFS-backed unit store that generates
// indexes to provide efficient lookups.
//
// It wraps a flatFileUnitStore and intercepts calls to Def, Defs,
// Refs, etc., using its indexes to satisfy those queries efficiently
// where possible. Otherwise it invokes the underlying store to
// perform a full scan.
type indexedUnitStore struct {
	// indexes is all of the indexes that should be built, written,
	// and read from. It contains all indexes for all types of data
	// (e.g., def indexes, ref indexes, etc.).
	indexes []interface{}

	*flatFileUnitStore
}

var _ UnitStore = (*indexedUnitStore)(nil)

// newIndexedUnitStore creates a new indexed unit store that stores
// data and indexes in fs.
func newIndexedUnitStore(fs rwvfs.FileSystem) UnitStoreImporter {
	return &indexedUnitStore{
		indexes: []interface{}{
			&defPathIndex{},
		},
		flatFileUnitStore: &flatFileUnitStore{fs: fs},
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
			if err := s.readIndex(px); err != nil {
				return nil, err
			}
		}
		ofs, err := dx.Defs(fs...)
		if err != nil {
			return nil, err
		}
		return s.defsAtOffsets(ofs)
	}

	return s.flatFileUnitStore.Defs(fs...)
}

// defsAtOffsets reads the defs at the given serialized byte offsets
// from the main def data file and returns them in arbitrary order.
func (s *indexedUnitStore) defsAtOffsets(ofs byteOffsets) ([]*graph.Def, error) {
	f, err := s.fs.Open(unitDefsFilename)
	if err != nil {
		return nil, err
	}
	defer func() {
		err2 := f.Close()
		if err == nil {
			err = err2
		}
	}()

	defs := make([]*graph.Def, len(ofs))
	for i, ofs := range ofs {
		if _, err := f.Seek(ofs, 0); err != nil {
			return nil, err
		}
		if err := Codec.Decode(f, &defs[i]); err != nil {
			return nil, err
		}
	}
	return defs, nil
}

// Refs implements UnitStore.
func (s *indexedUnitStore) Refs(fs ...RefFilter) ([]*graph.Ref, error) {
	// TODO(sqs): look up in ref indexes
	return s.flatFileUnitStore.Refs(fs...)
}

// Import calls to the underlying flatFileUnitStore to write the def
// and ref data files. It also builds and writes the indexes.
func (s *indexedUnitStore) Import(data graph.Output) error {
	cleanForImport(&data, "", "", "")

	// TODO(sqs): parallelize

	defOfs, err := s.flatFileUnitStore.writeDefs(&data)
	if err != nil {
		return err
	}
	if err := s.writeDefIndexes(&data, defOfs); err != nil {
		return err
	}

	refOfs, err := s.flatFileUnitStore.writeRefs(&data)
	if err != nil {
		return err
	}
	if err := s.writeRefIndexes(&data, refOfs); err != nil {
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
			if bx, ok := x.(indexBuilder); ok {
				par.Do(func() error {
					if err := bx.Build(data, ofs); err != nil {
						return err
					}
					if px, ok := x.(persistedIndex); ok {
						pf, err := s.fs.Create(fmt.Sprintf(indexFilename, px.Name()))
						if err != nil {
							return err
						}
						defer func() {
							err2 := pf.Close()
							if err == nil {
								err = err2
							}
						}()
						if err := px.Write(pf); err != nil {
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
func (s *indexedUnitStore) writeRefIndexes(data *graph.Output, ofs byteOffsets) error {
	// TODO(sqs): implement ref indexes
	return nil
}

// readIndex calls x.Read with the index's backing file.
func (s *indexedUnitStore) readIndex(x persistedIndex) (err error) {
	if x.Ready() {
		panic("x is already Ready; attempted to read it again")
	}

	f, err := s.fs.Open(fmt.Sprintf(indexFilename, x.Name()))
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
