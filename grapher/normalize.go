package grapher

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/sqs/fileset"

	"sourcegraph.com/sourcegraph/srclib/ann"
	"sourcegraph.com/sourcegraph/srclib/graph"
)

func fixOffsets(dir, filename string, fset *fileset.FileSet, files map[string]*fileset.File, offsets ...*uint32) {
	addOrGetFile := func(filename string) *fileset.File {
		if f, ok := files[filename]; ok {
			return f
		}
		data, err := ioutil.ReadFile(filename)
		if err != nil {
			panic("ReadFile " + filename + ": " + err.Error())
		}

		f := fset.AddFile(filename, fset.Base(), len(data))
		f.SetByteOffsetsForContent(data)
		files[filename] = f
		return f
	}

	defer func() {
		if e := recover(); e != nil {
			log.Printf("failed to convert unicode offset to byte offset in file %s (did grapher output a nonexistent byte offset?) continuing anyway...", filename)
		}
	}()
	if filename == "" {
		return
	}
	filename = filepath.Join(dir, filename)
	if fi, err := os.Stat(filename); err != nil || !fi.Mode().IsRegular() {
		return
	}
	f := addOrGetFile(filename)
	for _, offset := range offsets {
		if *offset == 0 {
			continue
		}
		*offset = uint32(f.ByteOffsetOfRune(int(*offset)))
	}
}

func ensureOffsetsAreByteOffsets(dir string, output *graph.Output) {
	fset := fileset.NewFileSet()
	files := make(map[string]*fileset.File)

	for _, s := range output.Defs {
		fixOffsets(dir, s.File, fset, files, &s.DefStart, &s.DefEnd)
	}
	for _, r := range output.Refs {
		fixOffsets(dir, r.File, fset, files, &r.Start, &r.End)
	}
	for _, d := range output.Docs {
		fixOffsets(dir, d.File, fset, files, &d.Start, &d.End)
	}
}

func sortedOutput(o *graph.Output) *graph.Output {
	sort.Sort(graph.Defs(o.Defs))
	sort.Sort(graph.Refs(o.Refs))
	sort.Sort(graph.Docs(o.Docs))
	sort.Sort(ann.Anns(o.Anns))
	return o
}

// NormalizeData sorts data and performs other postprocessing.
func NormalizeData(unitType, dir string, o *graph.Output) error {
	for _, ref := range o.Refs {
		if ref.DefRepo != "" {
			uri, err := graph.TryMakeURI(string(ref.DefRepo))
			if err != nil {
				return err
			}
			ref.DefRepo = uri
		}
		if ref.Repo != "" {
			uri, err := graph.TryMakeURI(string(ref.Repo))
			if err != nil {
				return err
			}
			ref.Repo = uri
		}
	}

	if unitType != "GoPackage" && unitType != "Dockerfile" && unitType != "NugetPackage" {
		ensureOffsetsAreByteOffsets(dir, o)
	}

	if err := ValidateRefs(o.Refs); err != nil {
		return err
	}
	if err := ValidateDefs(o.Defs); err != nil {
		return err
	}
	if err := ValidateDocs(o.Docs); err != nil {
		return err
	}

	sortedOutput(o)
	return nil
}
