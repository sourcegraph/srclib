package javascript

import (
	"encoding/json"
	"testing"

	"github.com/jmoiron/sqlx/types"
	"github.com/sourcegraph/srclib/graph"
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
			// qualify symbols with scope
			symbol: &graph.Symbol{
				Data: symbolDataJSON(symbolData{Key: DefPath{Path: "a.b"}}),
			},
			qual: graph.ScopeQualified,
			want: "a.b",
		},
		{
			// qualify file symbols with scope
			symbol: &graph.Symbol{
				Data: symbolDataJSON(symbolData{Key: DefPath{Namespace: "file", Path: "a.b.@local123.c.d"}}),
			},
			qual: graph.ScopeQualified,
			want: "c.d",
		},
		{
			// qualify symbols with module basename (dep-qualified)
			symbol: &graph.Symbol{
				Data: symbolDataJSON(symbolData{Key: DefPath{Path: "a.b", Module: "c/d"}}),
			},
			qual: graph.DepQualified,
			want: "d.a.b",
		},
		{
			// qualify symbols with pkg root and module (repository-wide)
			symbol: &graph.Symbol{
				SymbolKey: graph.SymbolKey{Unit: "x/y"},
				Data:      symbolDataJSON(symbolData{Key: DefPath{Path: "a.b", Module: "c/d"}}),
			},
			qual: graph.RepositoryWideQualified,
			want: "x/y/c/d.a.b",
		},
		{
			// qualify symbols with full path (lang-wide)
			symbol: &graph.Symbol{
				SymbolKey: graph.SymbolKey{Repo: "t/u", Unit: "x/y"},
				Data:      symbolDataJSON(symbolData{Key: DefPath{Path: "a.b", Module: "c/d"}}),
			},
			qual: graph.LanguageWideQualified,
			want: "t/u/x/y/c/d.a.b",
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
