package store

import (
	"io"

	"sourcegraph.com/sourcegraph/srclib/graph"
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

type indexBuilder interface {
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
