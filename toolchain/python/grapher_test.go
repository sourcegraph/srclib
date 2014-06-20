package python

import (
	"path/filepath"
	"reflect"
	"testing"

	"sourcegraph.com/sourcegraph/srcgraph/graph"
	"sourcegraph.com/sourcegraph/srcgraph/grapher2"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
)

func Test_GrapherTransform(t *testing.T) {
	p := defaultPythonEnv
	tests := []struct {
		TestName string
		Unit     unit.SourceUnit
		In       *rawGraphData
		Out      *grapher2.Output
		Err      error
	}{{
		TestName: "Well-formed input",
		Unit: &DistPackage{
			ProjectName:   "Pkg",
			RootDirectory: ".",
		},
		In: &rawGraphData{
			Graph: graphData_{
				Syms: []*pySym{{
					Path: filepath.Join(srcRoot, "pkg/module1/class/method1"),
					File: filepath.Join(srcRoot, "pkg/module1.py"),
				}},
				Refs: []*pyRef{{
					Sym:  filepath.Join(srcRoot, "pkg/module1/class/method1"),
					File: filepath.Join(srcRoot, "pkg/module2.py"),
				}, {
					Sym:  filepath.Join(p.sitePackagesDir(), "dep1/module"),
					File: filepath.Join(srcRoot, "pkg/module1.py"),
				}},
				Docs: []*pyDoc{},
			},
			Reqs: []requirement{{
				ProjectName: "Dep1",
				RepoURL:     "g.com/o/dep1",
				Packages:    []string{"dep1"},
				Modules:     []string{},
			}},
		},
		Out: &grapher2.Output{
			Symbols: []*graph.Symbol{{
				SymbolKey: graph.SymbolKey{
					Path:     "pkg/module1/class/method1",
					Unit:     "Pkg",
					UnitType: "PipPackage",
				},
				TreePath: "pkg/module1/class/method1",
				File:     "pkg/module1.py",
			}},
			Refs: []*graph.Ref{{
				SymbolUnitType: "PipPackage",
				SymbolUnit:     "Pkg",
				SymbolPath:     "pkg/module1/class/method1",
				UnitType:       "PipPackage",
				Unit:           "Pkg",
				File:           "pkg/module2.py",
			}, {
				SymbolRepo:     "g.com/o/dep1",
				SymbolUnitType: "PipPackage",
				SymbolUnit:     "Dep1",
				SymbolPath:     "dep1/module",
				UnitType:       "PipPackage",
				Unit:           "Pkg",
				File:           "pkg/module1.py",
			}},
		},
	}, {
		TestName: "Ignore requirement with no clone URL",
		Unit: &DistPackage{
			ProjectName:   "Pkg",
			RootDirectory: ".",
		},
		In: &rawGraphData{
			Graph: graphData_{
				Refs: []*pyRef{{
					Sym:  filepath.Join(p.sitePackagesDir(), "dep1/module"),
					File: filepath.Join(srcRoot, "pkg/module1.py"),
				}},
				Docs: []*pyDoc{},
			},
			Reqs: []requirement{{
				ProjectName: "Dep1",
				RepoURL:     "", // empty clone URL
				Packages:    []string{"dep1"},
				Modules:     []string{},
			}},
		},
		Out: &grapher2.Output{},
	}}

	for _, test := range tests {
		out, err := p.grapherTransform(test.In, test.Unit)
		if test.Err != nil {
			if test.Err != err {
				t.Errorf("Expected error %v, but got %v", test.Err, err)
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error %v", err)
				continue
			}

			// normalize output
			if len(out.Docs) == 0 {
				out.Docs = nil
			}
			if len(out.Refs) == 0 {
				out.Refs = nil
			}
			if len(out.Symbols) == 0 {
				out.Symbols = nil
			}
			// Ignore data field
			for _, sym := range out.Symbols {
				sym.Data = nil
			}

			if !reflect.DeepEqual(test.Out.Symbols, out.Symbols) {
				t.Errorf(`Test "%s": Expected output symbols %+v but got %+v`, test.TestName, test.Out.Symbols, out.Symbols)
			}
			if !reflect.DeepEqual(test.Out.Refs, out.Refs) {
				var expRefs, actRefs []graph.Ref
				for _, ref := range test.Out.Refs {
					expRefs = append(expRefs, *ref)
				}
				for _, ref := range out.Refs {
					actRefs = append(actRefs, *ref)
				}
				t.Errorf(`Test "%s": Expected output references %#v but got %#v`, test.TestName, expRefs, actRefs)
			}
			if !reflect.DeepEqual(test.Out.Docs, out.Docs) {
				t.Errorf(`Test: "%s": Expected output docs %+v but got %+v`, test.TestName, test.Out.Docs, out.Docs)
			}

		}
	}
}
