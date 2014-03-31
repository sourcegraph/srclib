package gog

import (
	"strings"
	"testing"
)

func TestPaths(t *testing.T) {
	cases := []struct {
		defs          string
		wantPaths     []defPath
		dontWantPaths []defPath
	}{
		{`type A struct {x string}`, []defPath{{"foo", "A/x"}}, nil},
		{`type A struct {b B};type B struct { c string }`, []defPath{{"foo", "A/b"}, {"foo", "A/b"}}, nil},
		{`type A struct {b struct { c string }}`, []defPath{{"foo", "A/b"}, {"foo", "A/b/c"}}, nil},
		{`type A struct { B }; type B struct { c string }`, []defPath{{"foo", "A/B"}}, []defPath{{"foo", "A/B/c"}, {"foo", "A/c"}}},
		{`type A struct { *B }; type B struct { c string }`, []defPath{{"foo", "A/B"}}, []defPath{{"foo", "A/B/c"}, {"foo", "A/c"}}},
		{`func _() { var a int; _ = a }`, []defPath{{"foo", "_/a"}}, nil},
		{`type A int; func (a A) x() { var b int; _ = b }`, []defPath{{"foo", "A/x/a"}, {"foo", "A/x/b"}}, nil},
		{`func _() { if true { var a int; _ = a } }`, []defPath{{"foo", "_/$sources[0]0/$sources[0]0/a"}}, nil},
		{`type A int; func (a A) F() {}`, []defPath{{"foo", "A/F"}}, nil},
		{`type A int; func (a *A) F() {}`, []defPath{{"foo", "A/F"}}, nil},
		{`func F() {f := func(a int) (b int) { c := 7; return c; }; _ = f }`, []defPath{{"foo", "F/f"}, {"foo", "F/$sources[0]0/a"}, {"foo", "F/$sources[0]0/b"}, {"foo", "F/$sources[0]0/c"}}, nil},
		{`func F() { {a:=0;_=a};{a:=0;_=a} }`, []defPath{{"foo", "F/$sources[0]0/a"}, {"foo", "F/$sources[0]1/a"}}, nil},
		{`func init() {}; func init() {}`, []defPath{{"foo", "init$sources[0]28"}, {"foo", "init$sources[0]44"}}, nil},
		{`var x struct { y int }`, []defPath{{"foo", "x/y"}}, nil},
		{`func f(x struct{y int}) { _ = x.y }`, []defPath{{"foo", "f/x/y"}}, nil},
		{`type I interface { A(); B() }`, []defPath{{"foo", "I"}, {"foo", "I/A"}, {"foo", "I/B"}}, nil},
		{`type I interface { A(x int); B(x int) }`, []defPath{{"foo", "I/A/x"}, {"foo", "I/B/x"}}, nil},
		{`type f func(i int); type g func(i int)`, []defPath{{"foo", "$sources[0]/$sources[0]0/i"}, {"foo", "$sources[0]/$sources[0]1/i"}}, nil},
	}

	for _, c := range cases {
		src := `package foo; /*START*/ ` + c.defs + ` /*END*/`
		start, end := strings.Index(src, "/*START*/"), strings.Index(src, "/*END*/")
		prog := createPkg(t, "foo", []string{src}, nil)

		g := New(prog)
		g.SkipDocs = true
		err := g.Graph(prog.Created[0])
		if err != nil {
			t.Fatal(err)
		}

		var paths []defPath
		for _, s := range g.Symbols {
			if s.IdentSpan[0] >= start && s.IdentSpan[1] <= end {
				paths = append(paths, s.SymbolKey.defPath())
			}
		}

		var printAllPaths bool
		for _, wantPath := range c.wantPaths {
			var found bool
			for _, path := range paths {
				if path == wantPath {
					found = true
				}
			}
			if !found {
				t.Errorf("%q: path not found: %+v", c.defs, wantPath)
				printAllPaths = true
			}
		}
		for _, dontWantPath := range c.dontWantPaths {
			for _, path := range paths {
				if path == dontWantPath {
					t.Errorf("%q: unwanted path: %+v", c.defs, dontWantPath)
					printAllPaths = true
					break
				}
			}
		}

		if printAllPaths {
			t.Logf("\n### Code:\n%s\n### All paths:", src)
			for _, path := range paths {
				t.Logf("  %+v", path)
			}
		}
	}
}
