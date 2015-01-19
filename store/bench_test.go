package store

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"testing"

	"sourcegraph.com/sourcegraph/rwvfs"
	"sourcegraph.com/sourcegraph/srclib/graph"
	"sourcegraph.com/sourcegraph/srclib/unit"
)

var (
	numVersions = flag.Int("bench.versions", 1, "number of versions (each of which has the denoted number of units & defs)")
	numUnits    = flag.Int("bench.units", 1, "number of source units (each of which has the denoted number of defs)")
	numFiles    = flag.Int("bench.files", 25, "number of distinct files (Def.File and Ref.File values)")
	numRefDefs  = flag.Int("bench.refdefs", 100, "number of distinct defs that refs point to")

	codec = flag.String("codec", "json", "flat file codec")
)

func BenchmarkFlatFile_Def1(b *testing.B)     { benchmarkDef(b, repoStore(), 1) }
func BenchmarkFlatFile_Def500(b *testing.B)   { benchmarkDef(b, repoStore(), 500) }
func BenchmarkFlatFile_Def5000(b *testing.B)  { benchmarkDef(b, repoStore(), 5000) }
func BenchmarkFlatFile_Def50000(b *testing.B) { benchmarkDef(b, repoStore(), 50000) }

func BenchmarkFlatFile_DefsByFile1(b *testing.B)     { benchmarkDefsByFile(b, repoStore(), 1) }
func BenchmarkFlatFile_DefsByFile500(b *testing.B)   { benchmarkDefsByFile(b, repoStore(), 500) }
func BenchmarkFlatFile_DefsByFile5000(b *testing.B)  { benchmarkDefsByFile(b, repoStore(), 5000) }
func BenchmarkFlatFile_DefsByFile50000(b *testing.B) { benchmarkDefsByFile(b, repoStore(), 50000) }

func BenchmarkFlatFile_RefsByFile1(b *testing.B)     { benchmarkRefsByFile(b, repoStore(), 1) }
func BenchmarkFlatFile_RefsByFile500(b *testing.B)   { benchmarkRefsByFile(b, repoStore(), 500) }
func BenchmarkFlatFile_RefsByFile5000(b *testing.B)  { benchmarkRefsByFile(b, repoStore(), 5000) }
func BenchmarkFlatFile_RefsByFile50000(b *testing.B) { benchmarkRefsByFile(b, repoStore(), 50000) }

func BenchmarkFlatFile_RefsByDefPath1(b *testing.B)     { benchmarkRefsByDefPath(b, repoStore(), 1) }
func BenchmarkFlatFile_RefsByDefPath500(b *testing.B)   { benchmarkRefsByDefPath(b, repoStore(), 500) }
func BenchmarkFlatFile_RefsByDefPath5000(b *testing.B)  { benchmarkRefsByDefPath(b, repoStore(), 5000) }
func BenchmarkFlatFile_RefsByDefPath50000(b *testing.B) { benchmarkRefsByDefPath(b, repoStore(), 50000) }

func repoStore() RepoStoreImporter {
	tmpDir, err := ioutil.TempDir("", "srclib-FlatFileRepoStore-bench")
	if err != nil {
		panic(err)
	}
	fs := rwvfs.OS(tmpDir)
	setCreateParentDirs(fs)

	var conf FlatFileConfig
	switch *codec {
	case "gob-json-gzip":
		conf.Codec = GobAndJSONGzipCodec{}
	case "gob-json":
		conf.Codec = GobAndJSONCodec{}
	case "json":
		conf.Codec = JSONCodec{}
	case "gob":
		conf.Codec = GobCodec{}
	default:
		fmt.Fprintln(os.Stderr, "Unknown -codec:", *codec)
		os.Exit(1)
	}

	return NewFlatFileRepoStore(fs, &conf)
}

func insertDefs(b *testing.B, rs RepoStoreImporter, numDefs int) {
	for v := 0; v < *numVersions; v++ {
		version := &Version{CommitID: fmt.Sprintf("commit%d", v)}
		for u := 0; u < *numUnits; u++ {
			unit := &unit.SourceUnit{Name: fmt.Sprintf("unit%d", u), Type: fmt.Sprintf("type%d", u)}
			data := graph.Output{Defs: make([]*graph.Def, numDefs)}
			for d := 0; d < numDefs; d++ {
				data.Defs[d] = &graph.Def{
					DefKey: graph.DefKey{Path: fmt.Sprintf("path%d", d)},
					Name:   fmt.Sprintf("name%d", d),
					File:   fmt.Sprintf("file%d", d%*numFiles),
				}
			}
			if err := rs.Import(version.CommitID, unit, data); err != nil {
				b.Fatal(err)
			}
		}
	}
}

func insertRefs(b *testing.B, rs RepoStoreImporter, numRefs int) {
	for v := 0; v < *numVersions; v++ {
		version := &Version{CommitID: fmt.Sprintf("commit%d", v)}
		for u := 0; u < *numUnits; u++ {
			unit := &unit.SourceUnit{Name: fmt.Sprintf("unit%d", u), Type: fmt.Sprintf("type%d", u)}
			data := graph.Output{Refs: make([]*graph.Ref, numRefs)}
			for r := 0; r < numRefs; r++ {
				data.Refs[r] = &graph.Ref{
					DefPath: fmt.Sprintf("path%d", r%*numRefDefs),
					File:    fmt.Sprintf("file%d", r%*numFiles),
					Start:   r % 1000,
					End:     (r + 7) % 1000,
				}
			}
			if err := rs.Import(version.CommitID, unit, data); err != nil {
				b.Fatal(err)
			}
		}
	}
}

func benchmarkDef(b *testing.B, rs RepoStoreImporter, numDefs int) {
	insertDefs(b, rs, numDefs)

	defKey := graph.DefKey{
		CommitID: fmt.Sprintf("commit%d", *numVersions/2),
		Unit:     fmt.Sprintf("unit%d", *numUnits/2),
		UnitType: fmt.Sprintf("type%d", *numUnits/2),
		Path:     fmt.Sprintf("path%d", numDefs/2),
	}

	checkCorrectness := false

	runtime.GC()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		def, err := rs.Def(defKey)
		if err != nil {
			b.Fatal(err)
		}
		if checkCorrectness {
			if def.DefKey != defKey {
				b.Fatalf("got DefKey %v, want %v", def.DefKey, defKey)
			}
		}
	}
}

func benchmarkDefsByFile(b *testing.B, rs RepoStoreImporter, numDefs int) {
	insertDefs(b, rs, numDefs)

	commitID := fmt.Sprintf("commit%d", *numVersions/2)
	defFilter := func(def *graph.Def) bool {
		return def.CommitID == commitID && def.File == "file0"
	}

	runtime.GC()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := rs.Defs(defFilter)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func benchmarkRefsByFile(b *testing.B, rs RepoStoreImporter, numRefs int) {
	insertRefs(b, rs, numRefs)

	commitID := fmt.Sprintf("commit%d", *numVersions/2)
	refFilter := func(ref *graph.Ref) bool {
		return ref.CommitID == commitID && ref.File == "file0"
	}

	runtime.GC()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := rs.Refs(refFilter)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func benchmarkRefsByDefPath(b *testing.B, rs RepoStoreImporter, numRefs int) {
	insertRefs(b, rs, numRefs)

	commitID := fmt.Sprintf("commit%d", *numVersions/2)
	refFilter := func(ref *graph.Ref) bool {
		return ref.CommitID == commitID && ref.DefPath == "path0"
	}

	runtime.GC()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := rs.Refs(refFilter)
		if err != nil {
			b.Fatal(err)
		}
	}
}
