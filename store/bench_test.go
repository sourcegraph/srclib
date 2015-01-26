package store

import (
	"flag"
	"fmt"
	"io/ioutil"
	"runtime"
	"testing"

	"sourcegraph.com/sourcegraph/rwvfs"
	"sourcegraph.com/sourcegraph/srclib/graph"
	"sourcegraph.com/sourcegraph/srclib/unit"
)

var (
	numVersions = flag.Int("bench.versions", 2, "number of versions (each of which has the denoted number of units & defs)")
	numUnits    = flag.Int("bench.units", 2, "number of source units (each of which has the denoted number of defs)")
	numPerFile  = flag.Int("bench.per-file", 25, "number of refs/defs/etc. per file")
	numRefDefs  = flag.Int("bench.refdefs", 10, "number of distinct defs that refs point to")
)

func BenchmarkFS_Def1(b *testing.B)     { benchmarkDef(b, repoStore(), 1) }
func BenchmarkFS_Def500(b *testing.B)   { benchmarkDef(b, repoStore(), 500) }
func BenchmarkFS_Def5000(b *testing.B)  { benchmarkDef(b, repoStore(), 5000) }
func BenchmarkFS_Def50000(b *testing.B) { benchmarkDef(b, repoStore(), 50000) }

func BenchmarkFS_DefsByFile1(b *testing.B)     { benchmarkDefsByFile(b, repoStore(), 1) }
func BenchmarkFS_DefsByFile500(b *testing.B)   { benchmarkDefsByFile(b, repoStore(), 500) }
func BenchmarkFS_DefsByFile5000(b *testing.B)  { benchmarkDefsByFile(b, repoStore(), 5000) }
func BenchmarkFS_DefsByFile50000(b *testing.B) { benchmarkDefsByFile(b, repoStore(), 50000) }

func BenchmarkFS_RefsByFile1(b *testing.B)     { benchmarkRefsByFile(b, repoStore(), 1) }
func BenchmarkFS_RefsByFile500(b *testing.B)   { benchmarkRefsByFile(b, repoStore(), 500) }
func BenchmarkFS_RefsByFile5000(b *testing.B)  { benchmarkRefsByFile(b, repoStore(), 5000) }
func BenchmarkFS_RefsByFile50000(b *testing.B) { benchmarkRefsByFile(b, repoStore(), 50000) }

func BenchmarkFS_RefsByFile1_filterFunc(b *testing.B) {
	benchmarkRefsByFile_filterFunc(b, repoStore(), 1)
}
func BenchmarkFS_RefsByFile500_filterFunc(b *testing.B) {
	benchmarkRefsByFile_filterFunc(b, repoStore(), 500)
}
func BenchmarkFS_RefsByFile5000_filterFunc(b *testing.B) {
	benchmarkRefsByFile_filterFunc(b, repoStore(), 5000)
}
func BenchmarkFS_RefsByFile50000_filterFunc(b *testing.B) {
	benchmarkRefsByFile_filterFunc(b, repoStore(), 50000)
}

func BenchmarkFS_RefsByDefPath1(b *testing.B)     { benchmarkRefsByDefPath(b, repoStore(), 1) }
func BenchmarkFS_RefsByDefPath500(b *testing.B)   { benchmarkRefsByDefPath(b, repoStore(), 500) }
func BenchmarkFS_RefsByDefPath5000(b *testing.B)  { benchmarkRefsByDefPath(b, repoStore(), 5000) }
func BenchmarkFS_RefsByDefPath50000(b *testing.B) { benchmarkRefsByDefPath(b, repoStore(), 50000) }

func repoStore() RepoStoreImporter {
	tmpDir, err := ioutil.TempDir("", "srclib-FSRepoStore-bench")
	if err != nil {
		panic(err)
	}
	fs := rwvfs.OS(tmpDir)
	setCreateParentDirs(fs)
	useIndexedStore = true
	return NewFSRepoStore(fs)
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
					File:   fmt.Sprintf("file%d", d%(1 + numDefs / *numPerFile)),
				}
				addSourceUnitFiles(unit, data.Defs[d].File)
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
					File:    fmt.Sprintf("file%d", r%(1 + numRefs / *numPerFile)),
					Start:   r % 1000,
					End:     (r + 7) % 1000,
				}
				addSourceUnitFiles(unit, data.Refs[r].File)
			}
			if err := rs.Import(version.CommitID, unit, data); err != nil {
				b.Fatal(err)
			}
		}
	}
}

func addSourceUnitFiles(u *unit.SourceUnit, file string) {
	for _, f := range u.Files {
		if f == file {
			return
		}
	}
	u.Files = append(u.Files, file)
}

func benchmarkDef(b *testing.B, rs RepoStoreImporter, numDefs int) {
	insertDefs(b, rs, numDefs)

	defKey := graph.DefKey{
		Repo:     "r", // dummy, must be filled in
		CommitID: fmt.Sprintf("commit%d", *numVersions/2),
		Unit:     fmt.Sprintf("unit%d", *numUnits/2),
		UnitType: fmt.Sprintf("type%d", *numUnits/2),
		Path:     fmt.Sprintf("path%d", numDefs/2),
	}

	checkCorrectness := false

	runtime.GC()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		defs, err := rs.Defs(ByDefKey(defKey))
		if err != nil {
			b.Fatal(err)
		}
		if len(defs) == 0 {
			b.Fatalf("not found: %v", defKey)
		}
		def := defs[0]
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
	defFilter := []DefFilter{
		ByCommitID(commitID),
		ByFiles("file0"),
	}

	runtime.GC()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		defs, err := rs.Defs(defFilter...)
		if err != nil {
			b.Fatal(err)
		}
		if len(defs) == 0 {
			b.Fatalf("no results: %v", defFilter)
		}
	}
}

func benchmarkRefsByFile(b *testing.B, rs RepoStoreImporter, numRefs int) {
	insertRefs(b, rs, numRefs)

	commitID := fmt.Sprintf("commit%d", *numVersions/2)
	refFilter := []RefFilter{
		ByCommitID(commitID),
		ByFiles("file0"),
	}

	runtime.GC()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		refs, err := rs.Refs(refFilter...)
		if err != nil {
			b.Fatal(err)
		}
		if len(refs) == 0 {
			b.Fatalf("no results: %v", refFilter)
		}
	}
}

func benchmarkRefsByFile_filterFunc(b *testing.B, rs RepoStoreImporter, numRefs int) {
	insertRefs(b, rs, numRefs)

	commitID := fmt.Sprintf("commit%d", *numVersions/2)
	refFilter := RefFilterFunc(func(ref *graph.Ref) bool {
		return ref.CommitID == commitID && ref.File == "file0"
	})

	runtime.GC()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		refs, err := rs.Refs(refFilter)
		if err != nil {
			b.Fatal(err)
		}
		if len(refs) == 0 {
			b.Fatalf("no results: %v", refFilter)
		}
	}
}

func benchmarkRefsByDefPath(b *testing.B, rs RepoStoreImporter, numRefs int) {
	insertRefs(b, rs, numRefs)

	commitID := fmt.Sprintf("commit%d", *numVersions/2)
	refFilter := []RefFilter{
		ByCommitID(commitID),
		ByRefDef(graph.RefDefKey{DefPath: "path0"}),
	}

	runtime.GC()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		refs, err := rs.Refs(refFilter...)
		if err != nil {
			b.Fatal(err)
		}
		if len(refs) == 0 {
			b.Fatalf("no results: %v", refFilter)
		}
	}
}
