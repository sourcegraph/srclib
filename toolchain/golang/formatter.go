package golang

import (
	"encoding/json"
	"path/filepath"

	"strings"

	"sourcegraph.com/sourcegraph/srcgraph/graph"
	"sourcegraph.com/sourcegraph/srcgraph/toolchain/golang/gog"
)

func init() {
	graph.RegisterSymbolFormatter(goPackageUnitType, symbolFormatter{})
}

type symbolFormatter struct{}

func (_ symbolFormatter) getData(s *graph.Symbol) *gog.SymbolInfo {
	var si gog.SymbolInfo
	if err := json.Unmarshal(s.Data, &si); err != nil {
		panic("unmarshal Go symbol data: " + err.Error())
	}
	return &si
}

func (_ symbolFormatter) LanguageName(s *graph.Symbol) string { return "Go" }

func (_ symbolFormatter) KindName(s *graph.Symbol) string {
	return s.SpecificKind
}

func (_ symbolFormatter) QualifiedName(s *graph.Symbol, relativeTo *graph.SymbolKey) string {
	return s.SpecificPath
}

func (f symbolFormatter) TypeString(s *graph.Symbol) string {
	si := f.getData(s)
	var ts string
	switch s.Kind {
	case graph.Func:
		ts = si.TypeString
		ts = strings.TrimPrefix(ts, "func")
	case graph.Type:
		ts = si.UnderlyingTypeString
		if i := strings.Index(ts, "{"); i != -1 {
			ts = ts[:i]
		}
		ts = " " + ts
	default:
		ts = " " + si.TypeString
	}
	ts = strings.Replace(ts, filepath.Join(string(s.Repo), s.Unit)+".", "", -1)
	return ts
}

func (_ symbolFormatter) RepositoryListing(pkglike *graph.Symbol) graph.RepositoryListingSymbol {
	// TODO(sqs)
	return graph.RepositoryListingSymbol{}
}
