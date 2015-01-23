package store

import (
	"io"

	"sourcegraph.com/sourcegraph/srclib/graph"
	"sourcegraph.com/sourcegraph/srclib/unit"
)

// A persistedIndex is an index that can be serialized and
// deserialized.
type persistedIndex interface {
	// Write serializes an index to a writer. The index's Read method
	// can be called to deserialize the index at a later date.
	Write(io.Writer) error

	// Read populates an index from a reader that contains the same
	// data that the index previously wrote (using Write).
	Read(io.Reader) error

	// Ready indicates whether the index is ready to be
	// queried. Persisted indexes typically become ready after their
	// Read method is called and returns.
	Ready() bool

	// Name is an identifier for this index that is unique in the
	// source unit. It is used to construct the name of the file that
	// this index is persisted to.
	Name() string
}

type byteOffsets []int64

type graphIndexBuilder interface {
	Build(*graph.Output, byteOffsets) error
}

type defIndex interface {
	// Covers returns the number of filters that this index
	// covers. Indexes with greater coverage are selected over others
	// with lesser coverage.
	Covers([]DefFilter) int

	// Defs returns the byte offsets (within the def data file) of the
	// defs that match the def filters.
	Defs(...DefFilter) (byteOffsets, error)
}

// bestCoverageDefIndex returns the index that has the greatest
// coverage for the given def filters, or nil if no indexes have any coverage.
func bestCoverageDefIndex(defIndexes []interface{}, fs []DefFilter) defIndex {
	maxCov := 0
	var maxX defIndex
	for _, x := range defIndexes {
		if x, ok := x.(defIndex); ok {
			cov := x.Covers(fs)
			if cov > maxCov {
				maxCov = cov
				maxX = x
			}
		}
	}
	return maxX
}

// fileByteRanges maps from filename to the byte ranges in a byte
// array that pertain to that file. It's used to index into the ref
// data to quickly read all of the refs in a given file.
type fileByteRanges map[string]byteRanges

// byteRanges' encodes the byte offsets of multiple objects. The first
// element is the byte offset within a file. Subsequent elements are
// the byte length of each object in the file.
type byteRanges []int64

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
	// Covers returns the number of filters that this index
	// covers. Indexes with greater coverage are selected over others
	// with lesser coverage.
	Covers([]RefFilter) int

	// Refs returns the byte ranges (in the ref data file) of matching
	// refs, or errNotIndexed if no indexes can be used to satisfy the
	// query.
	Refs(...RefFilter) ([]byteRanges, error)

	// Build constructs the index.
	Build(*graph.Output, fileByteRanges) error
}

// bestCoverageRefIndex returns the index that has the greatest
// coverage for the given ref filters, or nil if no indexes have any coverage.
func bestCoverageRefIndex(refIndexes []interface{}, fs []RefFilter) refIndex {
	maxCov := 0
	var maxX refIndex
	for _, x := range refIndexes {
		if x, ok := x.(refIndex); ok {
			cov := x.Covers(fs)
			if cov > maxCov {
				maxCov = cov
				maxX = x
			}
		}
	}
	return maxX
}

type unitIndex interface {
	// Covers returns the number of filters that this index
	// covers. Indexes with greater coverage are selected over others
	// with lesser coverage.
	Covers([]UnitFilter) int

	// Units returns the unit IDs units that match the unit filters.
	Units(...UnitFilter) ([]unit.ID2, error)
}

type unitIndexBuilder interface {
	Build([]*unit.SourceUnit) error
}

// bestCoverageUnitIndex returns the index that has the greatest
// coverage for the given unit filters, or nil if no indexes have any coverage.
func bestCoverageUnitIndex(unitIndexes []interface{}, fs []UnitFilter) unitIndex {
	maxCov := 0
	var maxX unitIndex
	for _, x := range unitIndexes {
		if x, ok := x.(unitIndex); ok {
			cov := x.Covers(fs)
			if cov > maxCov {
				maxCov = cov
				maxX = x
			}
		}
	}
	return maxX
}
