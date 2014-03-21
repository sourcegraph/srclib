package gog

import (
	"bytes"
	"flag"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"code.google.com/p/go.tools/go/loader"
)

var stdlibPath = flag.String("test.stdlib-pkg", "", "in TestStdlib, only graph this package (import path)")

// adapted from go/types stdlib_test.go

var (
	pkgCount         int // number of packages processed
	symCount         int
	refCount         int
	unresolvedIdents int
	start            time.Time
)

func TestStdlib(t *testing.T) {
	if data, err := exec.Command("go", "list", "std").Output(); err == nil {
		lines := bytes.Split(data, []byte{'\n'})
		start = time.Now()
		for _, line := range lines {
			if path := string(line); path != "" && !strings.HasPrefix(path, "cmd/") {
				if *stdlibPath == "" || path == *stdlibPath {
					testPkg(t, path)
				}
			}
		}
	} else {
		t.Fatal(err)
	}

	if testing.Verbose() {
		fmt.Println(pkgCount, "packages graphed in", time.Since(start))
		fmt.Printf("totals: %d symbols, %d refs\n", symCount, refCount)
		if unresolvedIdents > 0 {
			t.Logf("unresolved idents: %d", unresolvedIdents)
		}
	}
}

func testPkg(t *testing.T, path string) {
	if path == "unsafe" {
		return
	}
	conf := Default
	conf.SourceImports = *resolve
	conf.Import(path)
	prog, err := conf.Load()
	if err != nil {
		t.Fatal(path, err)
	}
	g := New(prog)

	start := time.Now()
	err = g.GraphAll()
	if err != nil {
		t.Fatal(err)
	}
	if testing.Verbose() {
		fmt.Printf("graphed %-22s\t% 4d msec   [% 6d symbols, % 6d refs]\n", path, time.Since(start)/time.Millisecond, len(g.Symbols), len(g.Refs))
	}
	pkgCount++
	symCount += len(g.Symbols)
	refCount += len(g.Refs)

	checkAllIdents(t, g, prog)
	checkUnique(t, g, prog)
}

func checkUnique(t *testing.T, g *Grapher, prog *loader.Program) {
	syms := make(map[defPath]*Symbol, len(g.Symbols))
	for _, s := range g.Symbols {
		key := s.SymbolKey.defPath()
		if x, present := syms[key]; present {
			t.Errorf("symbol %+v %s:%d-%d already defined at %s:%d-%d", key, s.File, s.IdentSpan[0], s.IdentSpan[1], x.File, x.IdentSpan[0], x.IdentSpan[1])
		} else {
			syms[key] = s
		}
	}
}
