package golang

import (
	"encoding/json"
	"testing"

	"github.com/jmoiron/sqlx/types"
	"sourcegraph.com/sourcegraph/srcgraph/graph"
	"sourcegraph.com/sourcegraph/srcgraph/toolchain/golang/gog"
)

func symbolInfo(si SymbolData) types.JsonText {
	b, err := json.Marshal(si)
	if err != nil {
		panic(err)
	}
	return b
}

func TestSymbolFormatter_Name(t *testing.T) {
	tests := []struct {
		symbol *graph.Symbol
		qual   graph.Qualification
		want   string
	}{
		{
			// unqualified
			symbol: &graph.Symbol{
				Name: "name",
				Data: types.JsonText(`{}`),
			},
			qual: graph.Unqualified,
			want: "name",
		},
		{
			// qualify methods with receiver
			symbol: &graph.Symbol{
				Name: "name",
				Data: symbolInfo(SymbolData{SymbolInfo: gog.SymbolInfo{Receiver: "*T", Kind: gog.Method}}),
			},
			qual: graph.ScopeQualified,
			want: "(*T).name",
		},
		{
			// all funcs are at pkg scope
			symbol: &graph.Symbol{
				Name: "name",
				Data: symbolInfo(SymbolData{SymbolInfo: gog.SymbolInfo{PkgName: "mypkg", Kind: gog.Func}}),
			},
			qual: graph.ScopeQualified,
			want: "name",
		},
		{
			// qualify funcs with pkg
			symbol: &graph.Symbol{
				Name: "Name",
				Data: symbolInfo(SymbolData{SymbolInfo: gog.SymbolInfo{PkgName: "mypkg", Kind: gog.Func}}),
			},
			qual: graph.DepQualified,
			want: "mypkg.Name",
		},
		{
			// qualify methods with receiver pkg
			symbol: &graph.Symbol{
				Name: "Name",
				Data: symbolInfo(SymbolData{SymbolInfo: gog.SymbolInfo{Receiver: "*T", PkgName: "mypkg", Kind: gog.Method}}),
			},
			qual: graph.DepQualified,
			want: "(*mypkg.T).Name",
		},
		{
			// qualify pkgs with import path relative to repo root
			symbol: &graph.Symbol{
				SymbolKey: graph.SymbolKey{Repo: "example.com/foo"},
				Name:      "subpkg",
				Kind:      "package",
				Data:      symbolInfo(SymbolData{PackageImportPath: "example.com/foo/mypkg/subpkg", SymbolInfo: gog.SymbolInfo{PkgName: "subpkg", Kind: gog.Package}}),
			},
			qual: graph.RepositoryWideQualified,
			want: "foo/mypkg/subpkg",
		},
		{
			// qualify funcs with import path
			symbol: &graph.Symbol{
				Name: "Name",
				Data: symbolInfo(SymbolData{PackageImportPath: "a/b", SymbolInfo: gog.SymbolInfo{PkgName: "x", Kind: gog.Func}}),
			},
			qual: graph.LanguageWideQualified,
			want: "a/b.Name",
		},
		{
			// qualify methods with receiver pkg
			symbol: &graph.Symbol{
				Name: "Name",
				Data: symbolInfo(SymbolData{PackageImportPath: "a/b", SymbolInfo: gog.SymbolInfo{Receiver: "*T", PkgName: "x", Kind: gog.Method}}),
			},
			qual: graph.LanguageWideQualified,
			want: "(*a/b.T).Name",
		},
		{
			// qualify pkgs with full import path
			symbol: &graph.Symbol{
				Name: "x",
				Data: symbolInfo(SymbolData{PackageImportPath: "a/b", SymbolInfo: gog.SymbolInfo{PkgName: "x", Kind: gog.Package}}),
			},
			qual: graph.LanguageWideQualified,
			want: "a/b",
		},
	}
	for _, test := range tests {
		sf := newSymbolFormatter(test.symbol)
		name := sf.Name(test.qual)
		if name != test.want {
			t.Errorf("%v qual %q: got %q, want %q", test.symbol, test.qual, name, test.want)
		}
	}
}
