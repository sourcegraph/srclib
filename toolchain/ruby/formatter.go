package ruby

import (
	"encoding/json"
	"strings"

	"sourcegraph.com/sourcegraph/srcgraph/graph"
)

func init() {
	graph.RegisterMakeSymbolFormatter(rubygemUnitType, newSymbolFormatter)
	graph.RegisterMakeSymbolFormatter(rubyLibUnitType, newSymbolFormatter)
}

func newSymbolFormatter(s *graph.Symbol) graph.SymbolFormatter {
	var si SymbolData
	if len(s.Data) > 0 {
		if err := json.Unmarshal(s.Data, &si); err != nil {
			panic("unmarshal Ruby symbol data: " + err.Error())
		}
	}
	return symbolFormatter{s, &si}
}

type symbolFormatter struct {
	symbol *graph.Symbol
	data   *SymbolData
}

func (f symbolFormatter) Language() string { return "JavaScript" }

func (f symbolFormatter) DefKeyword() string {
	switch f.data.RubyKind {
	case "method":
		return "def"
	case "class", "module":
		return f.data.RubyKind
	}
	return ""
}

func (f symbolFormatter) Kind() string { return f.data.RubyKind }

func (f symbolFormatter) Name(qual graph.Qualification) string {
	if f.data.isLocalVar() {
		return f.symbol.Name
	}

	switch qual {
	case graph.Unqualified:
		return f.symbol.Name
	case graph.ScopeQualified:
		return f.data.RubyPath
	case graph.DepQualified:
		return f.data.RubyPath
	case graph.RepositoryWideQualified:
		return f.data.RubyPath
	case graph.LanguageWideQualified:
		return f.data.RubyPath
	}
	panic("Name: unhandled qual " + string(qual))
}

func (f symbolFormatter) NameAndTypeSeparator() string {
	if f.data.RubyKind == "method" {
		return ""
	}
	return " "
}

func (f symbolFormatter) Type(qual graph.Qualification) string {
	var ts string
	if f.data.RubyKind == "method" {
		if i := strings.Index(f.data.Signature, "("); i != -1 {
			ts = f.data.Signature[i:]
		}
		ts += " " + f.data.ReturnType
	} else {
		ts = f.data.TypeString
	}
	return strings.TrimPrefix(ts, "::")
}
