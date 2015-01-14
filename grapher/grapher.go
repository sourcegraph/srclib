package grapher

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sqs/fileset"

	"sourcegraph.com/sourcegraph/srclib/ann"
	"sourcegraph.com/sourcegraph/srclib/config"
	"sourcegraph.com/sourcegraph/srclib/graph"
	"sourcegraph.com/sourcegraph/srclib/unit"
)

type Grapher interface {
	Graph(dir string, unit *unit.SourceUnit, c *config.Repository) (*graph.Output, error)
}

// TODO(sqs): add grapher validation of output

func ensureOffsetsAreByteOffsets(dir string, output *graph.Output) {
	fset := fileset.NewFileSet()
	files := make(map[string]*fileset.File)

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

	fix := func(filename string, offsets ...*int) {
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
			before, after := *offset, f.ByteOffsetOfRune(*offset)
			if before != after {
				log.Printf("Changed pos %d to %d in %s", before, after, filename)
			}
			*offset = f.ByteOffsetOfRune(*offset)
		}
	}

	for _, s := range output.Defs {
		fix(s.File, &s.DefStart, &s.DefEnd)
	}
	for _, r := range output.Refs {
		fix(r.File, &r.Start, &r.End)
	}
	for _, d := range output.Docs {
		fix(d.File, &d.Start, &d.End)
	}
	for _, a := range output.Anns {
		fix(a.File, &a.Start, &a.End)
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
func NormalizeData(currentRepoURI, unitType, dir string, o *graph.Output) error {
	for _, ref := range o.Refs {
		if ref.DefRepo != "" {
			ref.DefRepo = graph.MakeURI(string(ref.DefRepo))
		} else {
			ref.DefRepo = currentRepoURI
		}
		if ref.Repo == "" {
			ref.Repo = currentRepoURI
		}
	}

	if unitType != "GoPackage" && unitType != "Dockerfile" && !strings.HasPrefix(unitType, "Java") {
		ensureOffsetsAreByteOffsets(dir, o)
	}

	if err := ValidateRefs(o.Refs); err != nil {
		return err
	}
	if err := ValidateDefs(o.Defs); err != nil {
		return err
	}

	sortedOutput(o)
	return nil
}
