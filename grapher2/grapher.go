package grapher2

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/sqs/fileset"

	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/container"
	"sourcegraph.com/sourcegraph/srcgraph/graph"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
)

type Grapher interface {
	Graph(dir string, unit unit.SourceUnit, c *config.Repository) (*Output, error)
}

type Output struct {
	Symbols []*graph.Symbol `json:",omitempty"`
	Refs    []*graph.Ref    `json:",omitempty"`
	Docs    []*graph.Doc    `json:",omitempty"`
}

type GrapherBuilder interface {
	BuildGrapher(dir string, unit unit.SourceUnit, c *config.Repository) (*container.Command, error)
}

type DockerGrapher struct {
	GrapherBuilder
}

func (g DockerGrapher) Graph(dir string, unit unit.SourceUnit, c *config.Repository) (*Output, error) {
	cmd, err := g.BuildGrapher(dir, unit, c)
	if err != nil {
		return nil, err
	}

	if cmd == nil {
		// No container command returned; don't do anything.
		return &Output{}, nil
	}

	data, err := cmd.Run()
	if err != nil {
		return nil, err
	}

	var output *Output
	err = json.Unmarshal(data, &output)
	if err != nil {
		return nil, err
	}

	// Basic uniqueness checks.
	seenSymbolPaths := make(map[graph.SymbolPath]*graph.Symbol, len(output.Symbols))
	for _, s := range output.Symbols {
		if s0, seen := seenSymbolPaths[s.Path]; seen {
			return nil, fmt.Errorf("duplicate path in symbols output: %q\nsymbol 1: %+v\nsymbol 2: %+v", s.Path, s0, s)
		}
		seenSymbolPaths[s.Path] = s
	}

	return output, nil
}

// Graph uses the registered grapher (if any) to graph the source unit (whose repository is cloned to
// dir).
func Graph(dir string, u unit.SourceUnit, c *config.Repository) (*Output, error) {
	g, registered := Graphers[ptrTo(u)]
	if !registered {
		return nil, fmt.Errorf("no grapher registered for source unit %T", u)
	}

	o, err := g.Graph(dir, u, c)
	if err != nil {
		return nil, err
	}

	// If the grapher is known to output Unicode character offsets instead of
	// byte offsets, then convert all offsets to byte offsets.
	if ut := unit.Type(u); ut != "GoPackage" {
		ensureOffsetsAreByteOffsets(dir, o)
	}

	return sortedOutput(o), nil
}

func ensureOffsetsAreByteOffsets(dir string, output *Output) {
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
			*offset = f.ByteOffsetOfRune(*offset)
		}
	}

	for _, s := range output.Symbols {
		fix(s.File, &s.DefStart, &s.DefEnd)
	}
	for _, r := range output.Refs {
		fix(r.File, &r.Start, &r.End)
	}
	for _, d := range output.Docs {
		fix(d.File, &d.Start, &d.End)
	}
}

func sortedOutput(o *Output) *Output {
	sort.Sort(graph.Symbols(o.Symbols))
	sort.Sort(graph.Refs(o.Refs))
	sort.Sort(graph.Docs(o.Docs))
	return o
}
