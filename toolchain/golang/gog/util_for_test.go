package gog

import (
	"fmt"
	"go/ast"
	"io/ioutil"
	"testing"

	"code.google.com/p/go.tools/go/loader"
)

func graphPkgFromFiles(t *testing.T, path string, filenames []string) (*Grapher, *loader.Program) {
	prog := createPkgFromFiles(t, path, filenames)
	g := New(prog)
	err := g.GraphAll()
	if err != nil {
		t.Fatal(err)
	}
	return g, prog
}

func createPkgFromFiles(t *testing.T, path string, filenames []string) *loader.Program {
	sources := make([]string, len(filenames))
	for i, file := range filenames {
		src, err := ioutil.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		sources[i] = string(src)
	}
	return createPkg(t, path, sources, filenames)
}

func createPkg(t *testing.T, path string, sources []string, names []string) *loader.Program {
	conf := Default
	conf.SourceImports = *resolve

	var files []*ast.File
	for i, src := range sources {
		var name string
		if i < len(names) {
			name = names[i]
		} else {
			name = fmt.Sprintf("sources[%d]", i)
		}
		f, err := conf.ParseFile(name, src)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, f)
	}

	conf.CreateFromFiles(path, files...)
	prog, err := conf.Load()
	if err != nil {
		t.Fatal(err)
	}
	conf.Import("builtin")

	return prog
}
