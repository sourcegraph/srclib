package python

import (
	"encoding/json"
	"testing"

	"github.com/jmoiron/sqlx/types"
	"sourcegraph.com/sourcegraph/srcgraph/graph"
)

func symbolDataJSON(si symbolData) types.JsonText {
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
			// qualify symbols with scope (relative to file)
			symbol: &graph.Symbol{
				SymbolKey: graph.SymbolKey{Path: "a/b/c"},
				File:      "a/b.py",
			},
			qual: graph.ScopeQualified,
			want: "c",
		},
		{
			// qualify symbols with scope (relative to file)
			symbol: &graph.Symbol{
				SymbolKey: graph.SymbolKey{Path: "a/b/c"},
				File:      "a.py",
			},
			qual: graph.ScopeQualified,
			want: "b.c",
		},
		{
			// qualify symbols with module basename (dep-qualified)
			symbol: &graph.Symbol{
				SymbolKey: graph.SymbolKey{Path: "a/b/c/d"},
				File:      "a/b.py",
			},
			qual: graph.DepQualified,
			want: "b.c.d",
		},
		{
			// qualify symbols with pkg root and module (repository-wide)
			symbol: &graph.Symbol{
				SymbolKey: graph.SymbolKey{Path: "c/d"},
				File:      "a/b.py",
			},
			qual: graph.RepositoryWideQualified,
			want: "a.b.c.d",
		},
		{
			// qualify symbols with full path (lang-wide)
			symbol: &graph.Symbol{
				SymbolKey: graph.SymbolKey{Repo: "r/s", Path: "a/b/c/d"},
				File:      "a/b",
			},
			qual: graph.LanguageWideQualified,
			want: "r/s/a.b.c.d",
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
