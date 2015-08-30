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

type fileCache struct {
	files map[string]*fileset.File
	fset  *fileset.FileSet
}

func (c fileCache) addOrGet(filename string) *fileset.File {
	if f, ok := c.files[filename]; ok {
		return f
	}
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		panic("ReadFile " + filename + ": " + err.Error())
	}

	f := c.fset.AddFile(filename, c.fset.Base(), len(data))
	f.SetByteOffsetsForContent(data)
	f.SetLinesForContent(data)
	c.files[filename] = f
	return f
}

var files = fileCache{
	make(map[string]*fileset.File),
	fileset.NewFileSet(),
}

func ensureOffsetsAreByteOffsets(dir string, output *graph.Output) {
	fix := func(filename string, offsets ...*uint32) {
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
		f := files.addOrGet(filename)
		for _, offset := range offsets {
			if *offset == 0 {
				continue
			}
			before, after := *offset, uint32(f.ByteOffsetOfRune(int(*offset)))
			if before != after {
				log.Printf("Changed pos %d to %d in %s", before, after, filename)
			}
			*offset = uint32(f.ByteOffsetOfRune(int(*offset)))
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

func lineForOffset(dir, filename string, offset uint32) uint32 {
	if filename == "" {
		return 0
	}
	filename = filepath.Join(dir, filename)
	if fi, err := os.Stat(filename); err != nil || !fi.Mode().IsRegular() {
		return 0
	}
	f := files.addOrGet(filename)
	return uint32(f.Line(f.Pos(int(offset))))
}

// NormalizeData sorts data, adds line number to refs & defs and performs other
// postprocessing.
func NormalizeData(currentRepoURI, unitType, dir string, o *graph.Output) error {
	for _, ref := range o.Refs {
		if ref.DefRepo == currentRepoURI {
			ref.DefRepo = ""
		}
		if ref.DefRepo != "" {
			ref.DefRepo = graph.MakeURI(string(ref.DefRepo))
		}
		if ref.Repo == currentRepoURI {
			ref.Repo = ""
		}
		if ref.Repo != "" {
			ref.Repo = graph.MakeURI(string(ref.Repo))
		}
	}

	if unitType != "GoPackage" && unitType != "Dockerfile" && !strings.HasPrefix(unitType, "Java") {
		ensureOffsetsAreByteOffsets(dir, o)
	}

	for _, d := range o.Defs {
		d.StartLine = lineForOffset(dir, d.File, d.DefStart)
		d.EndLine = lineForOffset(dir, d.File, d.DefEnd)
	}
	for _, r := range o.Refs {
		r.StartLine = lineForOffset(dir, r.File, r.Start)
		r.EndLine = lineForOffset(dir, r.File, r.End)
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
