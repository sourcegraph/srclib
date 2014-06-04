package python

import (
	"encoding/json"
	"math/rand"
	"testing"

	"github.com/jmoiron/sqlx/types"
	"sourcegraph.com/sourcegraph/srcgraph/graph"
	"sourcegraph.com/sourcegraph/srcgraph/repo"
)

func TestSymbolFormatter_Name(t *testing.T) {
	tests := []struct {
		symbol *graph.Symbol
		want   map[graph.Qualification]string
	}{{
		symbol: symbolInfo{Name: "name"}.Symbol(),
		want:   map[graph.Qualification]string{graph.Unqualified: "name"},
	}, {
		symbol: symbolInfo{Repo: "g.com/o/r", TreePath: "a/b/c", File: "a/b.py"}.Symbol(),
		want: map[graph.Qualification]string{
			graph.ScopeQualified: "c", graph.DepQualified: "b.c", graph.RepositoryWideQualified: "a.b.c", graph.LanguageWideQualified: "g.com/o/r/a.b.c",
		},
	}, {
		symbol: symbolInfo{Repo: "g.com/o/r", TreePath: "a/b/c", File: "a.py"}.Symbol(),
		want: map[graph.Qualification]string{
			graph.ScopeQualified: "b.c", graph.DepQualified: "a.b.c", graph.RepositoryWideQualified: "a.b.c", graph.LanguageWideQualified: "g.com/o/r/a.b.c",
		},
	}, {
		symbol: symbolInfo{Repo: "g.com/o/r", TreePath: "aa/a/b", File: "aa/a.py"}.Symbol(),
		want: map[graph.Qualification]string{
			graph.ScopeQualified: "b", graph.DepQualified: "a.b", graph.RepositoryWideQualified: "aa.a.b", graph.LanguageWideQualified: "g.com/o/r/aa.a.b",
		},
	}, {
		symbol: symbolInfo{Repo: "g.com/o/r", TreePath: "a/b", File: "a/__init__.py"}.Symbol(),
		want: map[graph.Qualification]string{
			graph.ScopeQualified: "b", graph.DepQualified: "a.b", graph.RepositoryWideQualified: "a.b", graph.LanguageWideQualified: "g.com/o/r/a.b",
		},
	}}

	for _, test := range tests {
		sf := newSymbolFormatter(test.symbol)
		for qual, expName := range test.want {
			name := sf.Name(qual)
			if expName != name {
				t.Errorf("%v qual %q: want %q but got %q", test.symbol, qual, expName, name)
			}
		}
	}
}

func symbolDataJSON(si symbolData) types.JsonText {
	b, err := json.Marshal(si)
	if err != nil {
		panic(err)
	}
	return b
}

type symbolInfo struct {
	SID         graph.SID
	Repo        repo.URI
	CommitID    string
	UnitType    string
	Unit        string
	Path        string
	File        string
	Name        string
	TreePath    graph.TreePath
	NotExported bool
	Data        []byte
}

func (s symbolInfo) Symbol() *graph.Symbol {
	sid := s.SID
	if sid == 0 {
		sid = graph.SID(rand.Int63())
	}
	repo := s.Repo
	if repo == "" {
		repo = "r"
	}
	unitType := s.UnitType
	if unitType == "" {
		unitType = "t"
	}
	unit := s.Unit
	if unit == "" {
		unit = "u"
	}
	data := s.Data
	if data == nil {
		data = []byte(`{}`)
	}
	return &graph.Symbol{
		SID:       sid,
		SymbolKey: graph.SymbolKey{Repo: repo, CommitID: s.CommitID, UnitType: unitType, Unit: unit, Path: graph.SymbolPath(s.Path)},
		Name:      s.Name,
		File:      s.File,
		TreePath:  s.TreePath,
		Exported:  !s.NotExported,
		Data:      data,
	}
}
