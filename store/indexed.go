package store

import (
	"fmt"
	"io"
	"runtime"

	"code.google.com/p/rog-go/parallel"

	"sourcegraph.com/sourcegraph/rwvfs"
	"sourcegraph.com/sourcegraph/srclib/graph"
)

// An indexedUnitStore is a VFS-backed unit store that generates
// indexes to provide efficient lookups.
type indexedUnitStore struct {
	// fs is the filesystem where data and indexes are written to and
	// read from. The store may create multiple files and arbitrary
	// directory trees in fs (for indexes, etc.).
	fs rwvfs.FileSystem

	// indexes is all of the indexes that should be built, written,
	// and read from. It contains all indexes for all types of data
	// (e.g., def indexes, ref indexes, etc.).
	indexes []interface{}

	codec Codec
}

var _ UnitStore = (*indexedUnitStore)(nil)

// newIndexedUnitStore creates a new indexed unit store that stores
// data and indexes in fs and encodes data using the given codec.
func newIndexedUnitStore(fs rwvfs.FileSystem, c Codec) UnitStoreImporter {
	return &indexedUnitStore{
		fs: fs,
		indexes: []interface{}{
			&defPathIndex{},
		},
		codec: c,
	}
}

const (
	unitDefsFilename = "def.dat"
	unitRefsFilename = "ref.dat"
	indexFilename    = "%s.idx"
)

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
	// TODO(sqs): test failing when the index hasn't been Read yet - maybe this should check for persisted indexes that havent been read, and read them in
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

	// Otherwise, we must loop through all defs (which is slow).
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

	var defs []*graph.Def
	dec := newDecoder(s.codec, f)
	for {
		var def *graph.Def
		if err := dec.Decode(&def); err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		defs = append(defs, def)
	}
	return defs, nil
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
		if err := s.codec.Decode(f, &defs[i]); err != nil {
			return nil, err
		}
	}
	return defs, nil
}

// Refs implements UnitStore.
func (s *indexedUnitStore) Refs(fs ...RefFilter) ([]*graph.Ref, error) {
	// TODO(sqs): look up in ref indexes

	f, err := s.fs.Open(unitRefsFilename)
	if err != nil {
		return nil, err
	}
	defer func() {
		err2 := f.Close()
		if err == nil {
			err = err2
		}
	}()

	var allRefs []*graph.Ref
	if err := s.codec.Decode(f, &allRefs); err != nil {
		return nil, err
	}

	var refs []*graph.Ref // filtered refs
	for _, ref := range allRefs {
		if refFilters(fs).SelectRef(ref) {
			refs = append(refs, ref)
		}
	}

	return refs, nil
}

// Import implements UnitImporter.
func (s *indexedUnitStore) Import(data graph.Output) (err error) {
	cleanForImport(&data, "", "", "")
	// TODO(sqs): parallelize
	if err := s.writeDefs(&data); err != nil {
		return err
	}
	if err := s.writeRefs(&data); err != nil {
		return err
	}
	return nil
}

// writeDefs writes the main def data file and def indexes.
func (s *indexedUnitStore) writeDefs(data *graph.Output) error {
	ofs, err := s.writeDefData(data)
	if err != nil {
		return err
	}
	if err := s.writeDefIndexes(data, ofs); err != nil {
		return err
	}
	return nil
}

// writeDefdata writes the defs to the main def data file. It is not
// responsible for building or writing any indexes, but it returns the
// starting byte offset of each serialized def for indexes to use
// during index construction.
func (s *indexedUnitStore) writeDefData(data *graph.Output) (ofs byteOffsets, err error) {
	f, err := s.fs.Create(unitDefsFilename)
	if err != nil {
		return nil, err
	}
	defer func() {
		err2 := f.Close()
		if err == nil {
			err = err2
		}
	}()

	cw := &countingWriter{Writer: f}
	ofs = make(byteOffsets, len(data.Defs))
	for i, def := range data.Defs {
		ofs[i] = cw.n
		if err := s.codec.Encode(cw, def); err != nil {
			return nil, err
		}
	}
	return ofs, nil
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

// writeDefs writes the main ref data file.
func (s *indexedUnitStore) writeRefs(data *graph.Output) error {
	f, err := s.fs.Create(unitRefsFilename)
	if err != nil {
		return err
	}
	defer func() {
		err2 := f.Close()
		if err == nil {
			err = err2
		}
	}()
	return s.codec.Encode(f, data.Refs)
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

// countingWriter wraps an io.Writer, counting the number of bytes
// write.
type countingWriter struct {
	io.Writer
	n int64
}

func (cr *countingWriter) Write(p []byte) (n int, err error) {
	n, err = cr.Writer.Write(p)
	cr.n += int64(n)
	return
}
