package store

import (
	"encoding/binary"
	"errors"
	"io"

	"sourcegraph.com/sourcegraph/srclib/graph"
	"sourcegraph.com/sourcegraph/srclib/unit"
)

// An Index enables efficient store queries using filters that the
// index covers. An index may be in one of 3 states:
//
//  * Not built: the index neither exists in memory nor is it
//    persisted. It can't be used.
//
//  * Persisted but not ready: the index has been built and persisted
//    (e.g., to disk) but has not been loaded into memory and therefore
//    can't be used.
//
//  * Ready: the index is loaded into memory (either because it was
//    just built in memory, or because it was read from its persisted
//    form) and can be used.
type Index interface {
	// Ready indicates whether the index is ready to be
	// queried. Persisted indexes typically become ready after their
	// Read method is called and returns.
	Ready() bool

	// Covers returns the number of filters that this index
	// covers. Indexes with greater coverage are selected over others
	// with lesser coverage.
	Covers(filters interface{}) int
}

// A persistedIndex is an index that can be serialized and
// deserialized.
type persistedIndex interface {
	// Write serializes an index to a writer. The index's Read method
	// can be called to deserialize the index at a later date.
	Write(io.Writer) error

	// Read populates an index from a reader that contains the same
	// data that the index previously wrote (using Write).
	Read(io.Reader) error
}

type byteOffsets []int64

type defIndexBuilder interface {
	Build([]*graph.Def, byteOffsets) error
}

type defIndex interface {
	// Defs returns the byte offsets (within the def data file) of the
	// defs that match the def filters.
	Defs(...DefFilter) (byteOffsets, error)
}

// bestCoverageIndex returns the index that has the greatest coverage
// for the given filters, or nil if no indexes have any coverage. If
// test != nil, only indexes for which test(x) is true are considered.
func bestCoverageIndex(indexes map[string]Index, filters interface{}, test func(x interface{}) bool) (bestName string, best Index) {
	bestCov := 0
	for name, x := range indexes {
		if test != nil && !test(x) {
			continue
		}
		cov := x.Covers(filters)
		if cov > bestCov {
			bestCov = cov
			bestName = name
			best = x
		}
	}
	return bestName, best
}

func isUnitIndex(x interface{}) bool { _, ok := x.(unitIndex); return ok }
func isDefIndex(x interface{}) bool  { _, ok := x.(defIndex); return ok }
func isRefIndex(x interface{}) bool  { _, ok := x.(refIndex); return ok }

// fileByteRanges maps from filename to the byte ranges in a byte
// array that pertain to that file. It's used to index into the ref
// data to quickly read all of the refs in a given file.
type fileByteRanges map[string]byteRanges

// byteRanges' encodes the byte offsets of multiple objects. The first
// element is the byte offset within a file. Subsequent elements are
// the byte length of each object in the file.
type byteRanges []int64

// UnmarshalBinary implements encoding.BinaryUnmarshaler.
func (b *byteRanges) UnmarshalBinary(data []byte) error {
	for {
		v, n := binary.Varint(data)
		if n == 0 {
			break
		}
		if n < 0 {
			return errors.New("byteRanges varint error")
		}
		*b = append(*b, v)
		data = data[n:]
	}
	return nil
}

// MarshalBinary implements encoding.BinaryMarshaler.
func (b byteRanges) MarshalBinary() ([]byte, error) {
	data := make([]byte, len(b)*binary.MaxVarintLen64)
	var n int
	for _, v := range b {
		n += binary.PutVarint(data[n:], v)
	}
	return data[:n], nil
}

// start is the offset of the first byte of the first object, relative
// to the beginning of the file.
func (br byteRanges) start() int64 { return br[0] }

// byteRange's first element is the byte offset within a file, and its
// second element is number of bytes in the range.
type byteRange [2]int64

// refsByFileStartEnd sorts refs by (file, start, end).
type refsByFileStartEnd []*graph.Ref

func (v refsByFileStartEnd) Len() int { return len(v) }
func (v refsByFileStartEnd) Less(i, j int) bool {
	a, b := v[i], v[j]
	return a.File < b.File || (a.File == b.File && a.Start < b.Start) || (a.File == b.File && a.Start == b.Start && a.End < b.End)
}
func (v refsByFileStartEnd) Less2(i, j int) bool {
	a, b := v[i], v[j]
	if a.File == b.File {
		if a.Start == b.Start {
			return a.End < b.End
		}
		return a.Start < b.Start
	}
	return a.File < b.File
}
func (v refsByFileStartEnd) Swap(i, j int) { v[i], v[j] = v[j], v[i] }

type refIndex interface {
	// Refs returns the byte ranges (in the ref data file) of matching
	// refs, or errNotIndexed if no indexes can be used to satisfy the
	// query.
	Refs(...RefFilter) ([]byteRanges, error)
}

type refIndexBuilder interface {
	// Build constructs the index in memory.
	Build([]*graph.Ref, fileByteRanges) error
}

type unitIndex interface {
	// Units returns the unit IDs units that match the unit filters.
	Units(...UnitFilter) ([]unit.ID2, error)
}

type unitIndexBuilder interface {
	// Build constructs the index in memory.
	Build([]*unit.SourceUnit) error
}
