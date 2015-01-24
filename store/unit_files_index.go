package store

import (
	"encoding/json"
	"fmt"
	"io"
	"path"

	"github.com/alecthomas/mph"
	"sourcegraph.com/sourcegraph/srclib/unit"
)

// NOTE(sqs): There is a lot of duplication here with defFilesIndex.

// unitFilesIndex makes it fast to determine which source units
// contain a file (or files in a dir).
type unitFilesIndex struct {
	mph   *mph.CHD
	ready bool
}

var _ interface {
	persistedIndex
	unitIndexBuilder
	unitIndex
} = (*unitFilesIndex)(nil)

var c_unitFilesIndex_getByPath = 0 // counter

func (x *unitFilesIndex) String() string { return fmt.Sprintf("unitFilesIndex(ready=%v)", x.ready) }

// getByFile returns a list of source units that contain the file
// specified by the path. The path can also be a directory, in which
// case all source units that contain files underneath that directory
// are returned.
func (x *unitFilesIndex) getByPath(path string) ([]unit.ID2, bool, error) {
	vlog.Printf("unitFilesIndex.getByPath(%s)", path)
	c_unitFilesIndex_getByPath++

	if x.mph == nil {
		panic("mph not built/read")
	}
	v := x.mph.Get([]byte(path))
	if v == nil {
		return nil, false, nil
	}

	var us []unit.ID2
	if err := json.Unmarshal(v, &us); err != nil {
		return nil, true, err
	}
	return us, true, nil
}

// Covers implements unitIndex.
func (x *unitFilesIndex) Covers(fs []UnitFilter) int {
	cov := 0
	for _, f := range fs {
		if _, ok := f.(ByFilesFilter); ok {
			cov++
		}
	}
	return cov
}

// Units implements unitIndex.
func (x *unitFilesIndex) Units(fs ...UnitFilter) ([]unit.ID2, error) {
	for _, f := range fs {
		if ff, ok := f.(ByFilesFilter); ok {
			files := ff.ByFiles()
			umap := map[unit.ID2]struct{}{}
			for _, file := range files {
				u, _, err := x.getByPath(file)
				if err != nil {
					return nil, err
				}
				for _, uu := range u {
					umap[uu] = struct{}{}
				}
			}
			us := make([]unit.ID2, 0, len(umap))
			for u := range umap {
				us = append(us, u)
			}

			vlog.Printf("unitFilesIndex(%v): Found units %v using index.", fs, us)
			return us, nil
		}
	}
	return nil, nil
}

// Build implements unitIndexBuilder.
func (x *unitFilesIndex) Build(units []*unit.SourceUnit) error {
	b := mph.Builder()
	filesToUnits := make(map[string][]unit.ID2, len(units)*10)
	for _, u := range units {
		for _, f := range u.Files {
			f = path.Clean(f)
			filesToUnits[f] = append(filesToUnits[f], u.ID2())
		}
	}
	for file, fileUnits := range filesToUnits {
		ub, err := json.Marshal(fileUnits)
		if err != nil {
			return err
		}
		b.Add([]byte(file), ub)
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
func (x *unitFilesIndex) Write(w io.Writer) error {
	if x.mph == nil {
		panic("no mph to write")
	}
	return x.mph.Write(w)
}

// Read implements persistedIndex.
func (x *unitFilesIndex) Read(r io.Reader) error {
	var err error
	x.mph, err = mph.Read(r)
	x.ready = (err == nil)
	return err
}

// Ready implements persistedIndex.
func (x *unitFilesIndex) Ready() bool { return x.ready }

// Name implements persistedIndex.
func (x *unitFilesIndex) Name() string { return "file" }
