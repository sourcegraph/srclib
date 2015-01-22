package store

import (
	"io"
	"strconv"

	"github.com/alecthomas/mph"
	"sourcegraph.com/sourcegraph/srclib/graph"
)

type defPathIndex struct {
	mph   *mph.CHD
	ready bool
}

var _ interface {
	persistedIndex
	indexBuilder
	defIndex
} = (*defPathIndex)(nil)

func (x *defPathIndex) getByPath(defPath string) (int64, bool) {
	if x.mph == nil {
		panic("mph not built/read")
	}
	v := x.mph.Get([]byte(defPath))
	if v == nil {
		return 0, false
	}
	ofs, err := strconv.ParseInt(string(v), 36, 64)
	if err != nil {
		panic(err)
	}
	return ofs, true
}

func (x *defPathIndex) Def(key graph.DefKey) (int64, bool) {
	return x.getByPath(key.Path)
}

// Covers implements defIndex.
func (x *defPathIndex) Covers(fs []DefFilter) int {
	cov := 0
	for _, f := range fs {
		if _, ok := f.(ByDefPathFilter); ok {
			cov++
		}
	}
	return cov
}

// Defs implements defIndex.
func (x *defPathIndex) Defs(f ...DefFilter) (byteOffsets, error) {
	for _, ff := range f {
		if pf, ok := ff.(ByDefPathFilter); ok {
			ofs, found := x.getByPath(pf.ByDefPath())
			if !found {
				return nil, nil
			}
			return byteOffsets{ofs}, nil
		}
	}
	return nil, nil
}

// Build implements indexBuilder.
func (x *defPathIndex) Build(data *graph.Output, ofs byteOffsets) error {
	b := mph.Builder()
	for i, def := range data.Defs {
		b.Add([]byte(def.Path), []byte(strconv.FormatInt(ofs[i], 36)))
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
func (x *defPathIndex) Write(w io.Writer) error {
	if x.mph == nil {
		panic("no mph to write")
	}
	return x.mph.Write(w)
}

// Read implements persistedIndex.
func (x *defPathIndex) Read(r io.Reader) error {
	var err error
	x.mph, err = mph.Read(r)
	x.ready = (err == nil)
	return err
}

// Ready implements persistedIndex.
func (x *defPathIndex) Ready() bool { return x.ready }

// Name implements persistedIndex.
func (x *defPathIndex) Name() string { return "path" }
