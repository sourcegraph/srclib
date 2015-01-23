package store

import (
	"encoding/json"
	"io"

	"github.com/alecthomas/mph"
	"sourcegraph.com/sourcegraph/srclib/graph"
)

// refFileIndex makes it fast to determine which refs (within in a
// source unit) are in a file.
type refFileIndex struct {
	mph   *mph.CHD
	ready bool
}

var _ interface {
	persistedIndex
	refIndex
} = (*refFileIndex)(nil)

var c_refFileIndex_getByFile = 0 // counter

// getByFile returns a byteRanges describing the positions of refs in
// the given source file (i.e., for which ref.File == file). The
// byteRanges refer to offsets within the ref data file.
func (x *refFileIndex) getByFile(file string) (byteRanges, bool, error) {
	c_refFileIndex_getByFile++
	if x.mph == nil {
		panic("mph not built/read")
	}
	v := x.mph.Get([]byte(file))
	if v == nil {
		return nil, false, nil
	}

	// TODO(sqs): using JSON for this is really inefficient and stupid
	var br byteRanges
	if err := json.Unmarshal(v, &br); err != nil {
		return nil, true, err
	}
	return br, true, nil
}

// Covers implements defIndex.
func (x *refFileIndex) Covers(fs []RefFilter) int {
	// TODO(sqs): this index also covers RefStart/End range filters
	// (when those are added).
	cov := 0
	for _, f := range fs {
		if _, ok := f.(ByFileFilter); ok {
			cov++
		}
	}
	return cov
}

// Refs implements refIndex.
func (x *refFileIndex) Refs(fs ...RefFilter) ([]byteRanges, error) {
	for _, f := range fs {
		if ff, ok := f.(ByFileFilter); ok {
			br, found, err := x.getByFile(ff.ByFile())
			if !found || err != nil {
				return nil, err
			}
			return []byteRanges{br}, nil
		}
	}
	return nil, nil
}

// Build creates the refFileIndex.
func (x *refFileIndex) Build(data *graph.Output, fbr fileByteRanges) error {
	b := mph.Builder()
	for file, br := range fbr {
		v, err := json.Marshal(br)
		if err != nil {
			return err
		}
		b.Add([]byte(file), v)
	}
	h, err := b.Build()
	if err != nil {
		return err
	}
	x.mph = h
	x.ready = true
	return nil
}

// Write implements persistedIndex.
func (x *refFileIndex) Write(w io.Writer) error {
	if x.mph == nil {
		panic("no mph to write")
	}
	return x.mph.Write(w)
}

// Read implements persistedIndex.
func (x *refFileIndex) Read(r io.Reader) error {
	var err error
	x.mph, err = mph.Read(r)
	x.ready = (err == nil)
	return err
}

// Ready implements persistedIndex.
func (x *refFileIndex) Ready() bool { return x.ready }

// Name implements persistedIndex.
func (x *refFileIndex) Name() string { return "ref-file" }
