package store

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/alecthomas/mph"
	"sourcegraph.com/sourcegraph/srclib/graph"
)

// NOTE(sqs): There is a lot of duplication here with unitFilesIndex.

// defFilesIndex makes it fast to determine which source units
// contain a file (or files in a dir).
type defFilesIndex struct {
	// filters restricts the index to only indexing defs that pass all
	// of the filters.
	filters []DefFilter

	// perFile is the number of defs per file to index.
	perFile int

	mph   *mph.CHD
	ready bool
}

var _ interface {
	persistedIndex
	graphIndexBuilder
	defIndex
} = (*defFilesIndex)(nil)

var c_defFilesIndex_getByPath = 0 // counter

func (x *defFilesIndex) String() string {
	return fmt.Sprintf("defFilesIndex(ready=%v, filters=%v)", x.ready, x.filters)
}

// getByFile returns a list of source units that contain the file
// specified by the path. The path can also be a directory, in which
// case all source units that contain files underneath that directory
// are returned.
func (x *defFilesIndex) getByPath(path string) (byteOffsets, bool, error) {
	vlog.Printf("defFilesIndex.getByPath(%s)", path)
	c_defFilesIndex_getByPath++

	if x.mph == nil {
		panic("mph not built/read")
	}
	v := x.mph.Get([]byte(path))
	if v == nil {
		return nil, false, nil
	}

	var ofs byteOffsets
	if err := json.Unmarshal(v, &ofs); err != nil {
		return nil, true, err
	}
	return ofs, true, nil
}

// Covers implements defIndex.
func (x *defFilesIndex) Covers(fs []DefFilter) int {
	// TODO(sqs): ensure that x.filters is equivalent to fs (might
	// require an equals() method on filters, for filters with
	// internal state that we don't necessarily want to use when
	// testing equality). Otherwise this just assumes that fs has a
	// Limit, an Exported=true/Nonlocal=true filter, etc.
	cov := 0
	for _, f := range fs {
		if _, ok := f.(ByFilesFilter); ok {
			cov++
		}
	}
	return cov
}

// Defs implements defIndex.
func (x *defFilesIndex) Defs(fs ...DefFilter) (byteOffsets, error) {
	for _, f := range fs {
		if ff, ok := f.(ByFilesFilter); ok {
			files := ff.ByFiles()
			var allOfs byteOffsets
			for _, file := range files {
				ofs, _, err := x.getByPath(file)
				if err != nil {
					return nil, err
				}
				allOfs = append(allOfs, ofs...)
			}

			vlog.Printf("defFilesIndex(%v): Found %d def offsets using index.", fs, len(allOfs))
			return allOfs, nil
		}
	}
	return nil, nil
}

// Build implements graphIndexBuilder.
func (x *defFilesIndex) Build(data *graph.Output, ofs byteOffsets) error {
	b := mph.Builder()
	filesToDefOfs := make(map[string]byteOffsets, len(data.Defs)/50)
	for i, def := range data.Defs {
		if defFilters(x.filters).SelectDef(def) {
			if len(filesToDefOfs) < x.perFile {
				filesToDefOfs[def.File] = append(filesToDefOfs[def.File], ofs[i])
			}
		}
	}
	for file, defOfs := range filesToDefOfs {
		ob, err := json.Marshal(defOfs)
		if err != nil {
			return err
		}
		b.Add([]byte(file), ob)
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
func (x *defFilesIndex) Write(w io.Writer) error {
	if x.mph == nil {
		panic("no mph to write")
	}
	return x.mph.Write(w)
}

// Read implements persistedIndex.
func (x *defFilesIndex) Read(r io.Reader) error {
	var err error
	x.mph, err = mph.Read(r)
	x.ready = (err == nil)
	return err
}

// Ready implements persistedIndex.
func (x *defFilesIndex) Ready() bool { return x.ready }

// Name implements persistedIndex. TODO(sqs): add some string repr of
// x.filters to this name so we can have multiple such indexes.
func (x *defFilesIndex) Name() string { return "def-file" }
