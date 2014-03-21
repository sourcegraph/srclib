package gog

import (
	"flag"
	"go/ast"
	"go/token"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"code.google.com/p/go.tools/go/loader"
)

var identFile = flag.String("test.idents", "", "print out all idents in files whose name contains this substring")
var resolve = flag.Bool("test.resolve", false, "test that refs resolve to existing symbols")

func TestIdentCoverage(t *testing.T) {
	files, err := filepath.Glob("testdata/*.go")
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(files)

	g, prog := graphPkgFromFiles(t, "testdata", files)

	checkAllIdents(t, g, prog)
}

func (s *SymbolKey) defPath() defPath {
	return defPath{s.PackageImportPath, strings.Join(s.Path, "/")}
}

// checkAllIdents checks that every *ast.Ident has a corresponding Symbol or
// Ref.
func checkAllIdents(t *testing.T, g *Grapher, prog *loader.Program) {
	defs := make(map[defPath]struct{}, len(g.Symbols))
	byIdentPos := make(map[identPos]interface{}, len(g.Symbols)+len(g.Refs))
	for _, s := range g.Symbols {
		defs[s.SymbolKey.defPath()] = struct{}{}
		byIdentPos[identPos{s.File, s.IdentSpan[0], s.IdentSpan[1]}] = s
	}
	for _, r := range g.Refs {
		byIdentPos[identPos{r.File, r.Span[0], r.Span[1]}] = r
	}
	for _, pkg := range prog.Created {
		for _, f := range pkg.Files {
			printAll := *identFile != "" && strings.Contains(prog.Fset.Position(f.Name.Pos()).Filename, *identFile)
			checkIdents(t, prog.Fset, f, byIdentPos, defs, g, printAll)
			checkUnique(t, g, prog)
		}
	}
}

type defPath struct {
	pkg  string
	path string
}

type identPos struct {
	file       string
	start, end int
}

func checkIdents(t *testing.T, fset *token.FileSet, file *ast.File, idents map[identPos]interface{}, defs map[defPath]struct{}, g *Grapher, printAll bool) {
	ast.Inspect(file, func(n ast.Node) bool {
		if x, ok := n.(*ast.Ident); ok && !ignoreIdent(g, x) {
			pos, end := fset.Position(x.Pos()), fset.Position(x.End())
			if printAll {
				t.Logf("ident %q at %s:%d-%d", x.Name, pos.Filename, pos.Offset, end.Offset)
			}
			ip := identPos{pos.Filename, pos.Offset, end.Offset}
			if obj, present := idents[ip]; !present {
				t.Errorf("unresolved ident %q at %s", x.Name, pos)
			} else if ref, ok := obj.(*Ref); ok {
				if printAll {
					t.Logf("ref to %+v from ident %q at %s:%d-%d", ref.Symbol, x.Name, pos.Filename, pos.Offset, end.Offset)
				}
				if *resolve {
					if _, resolved := defs[ref.Symbol.defPath()]; !resolved && !ignoreRef(ref.Symbol.defPath()) {
						t.Errorf("unresolved ref %q to %+v at %s", x.Name, ref.Symbol.defPath(), pos)
						unresolvedIdents++
					}
				}
			}
			return false
		}
		return true
	})
}

func ignoreRef(dp defPath) bool {
	return dp.pkg == "builtin" || dp.pkg == "unsafe"
}

func ignoreIdent(g *Grapher, x *ast.Ident) bool {
	if x.Name == "_" {
		return true
	}
	if _, skip := g.skipResolve[x]; skip {
		return true
	}
	return false
}
