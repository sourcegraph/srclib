package store

import (
	"flag"
	"fmt"
	"runtime"
	"testing"

	"strings"

	"sourcegraph.com/sourcegraph/rwvfs"
	"sourcegraph.com/sourcegraph/srclib/graph"
)

var (
	numDefs = flag.Int("bench.defs", 500, "number of defs")
	numRefs = flag.Int("bench.refs", 500, "number of refs")
)

func BenchmarkFlatFileUnitStore_Def(b *testing.B) {
	useIndexedUnitStore = false
	benchmarkUnitStore_Def(b, ffUnitStore(), *numDefs)
}
func BenchmarkIndexedUnitStore_Def(b *testing.B) {
	useIndexedUnitStore = true
	benchmarkUnitStore_Def(b, idxUnitStore(), *numDefs)
}

func BenchmarkFlatFileUnitStore_Defs_all(b *testing.B) {
	useIndexedUnitStore = false
	benchmarkUnitStore_Defs_all(b, ffUnitStore(), *numDefs)
}
func BenchmarkIndexedUnitStore_Defs_all(b *testing.B) {
	useIndexedUnitStore = true
	benchmarkUnitStore_Defs_all(b, idxUnitStore(), *numDefs)
}

func BenchmarkFlatFileUnitStore_Defs_byFile(b *testing.B) {
	useIndexedUnitStore = false
	benchmarkUnitStore_Defs_byFile(b, ffUnitStore(), *numDefs)
}
func BenchmarkIndexedUnitStore_Defs_byFile(b *testing.B) {
	useIndexedUnitStore = true
	benchmarkUnitStore_Defs_byFile(b, idxUnitStore(), *numDefs)
}

func ffUnitStore() UnitStoreImporter {
	fs := rwvfs.Map(map[string]string{})
	return &flatFileUnitStore{fs: fs}
}

func idxUnitStore() UnitStoreImporter {
	fs := rwvfs.Map(map[string]string{})
	return newIndexedUnitStore(fs)
}

func benchmarkUnitStore_Def(b *testing.B, us UnitStoreImporter, numDefs int) {
	data := createUnitStoreBenchmarkData(b, numDefs, 0)
	if err := us.Import(data); err != nil {
		b.Fatal("Import:", err)
	}

	defKey := data.Defs[len(data.Defs)*93/99].DefKey

	runtime.GC()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		def, err := us.Def(defKey)
		if err != nil {
			b.Fatal(err)
		}
		if def.Path != defKey.Path {
			b.Fatalf("def paths do not match: got %q, want %q", def.Path, defKey.Path)
		}
	}
}

func benchmarkUnitStore_Defs_all(b *testing.B, us UnitStoreImporter, numDefs int) {
	data := createUnitStoreBenchmarkData(b, numDefs, 0)
	if err := us.Import(data); err != nil {
		b.Fatal("Import:", err)
	}

	runtime.GC()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := us.Defs()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func benchmarkUnitStore_Defs_byFile(b *testing.B, us UnitStoreImporter, numDefs int) {
	data := createUnitStoreBenchmarkData(b, numDefs, 0)
	if err := us.Import(data); err != nil {
		b.Fatal("Import:", err)
	}

	file := data.Defs[len(data.Defs)*93/99].File
	byFile := DefFilterFunc(func(def *graph.Def) bool {
		return def.File == file
	})

	runtime.GC()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := us.Defs(byFile)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func createUnitStoreBenchmarkData(b *testing.B, numDefs, numRefs int) graph.Output {
	data := graph.Output{
		Defs: make([]*graph.Def, numDefs),
		Refs: make([]*graph.Ref, numRefs),
	}
	kinds := []string{"aaaaa", "bbbbb", "ccccc", "ddddd", "eeeee"}
	for d := 0; d < numDefs; d++ {
		data.Defs[d] = &graph.Def{
			DefKey:   graph.DefKey{Path: fmt.Sprintf("parent%d/parent%d/child%d", d%(1+(numDefs/101)), d%(1+(numDefs/193)), d)},
			Name:     fmt.Sprintf("name%d", d),
			File:     fmt.Sprintf("file%d", d%(1+(numDefs/53))),
			Kind:     kinds[d%len(kinds)],
			DefStart: d % 10000,
			DefEnd:   (d % 10000) + (d % 15),
			Data:     []byte(`"` + strings.Repeat("x", 150) + `"`),
		}
	}
	return data
}
