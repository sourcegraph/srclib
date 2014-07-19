package javascript

import (
	"encoding/json"
	"path/filepath"

	"strings"

	"github.com/sourcegraph/srclib/graph"
)

func init() {
	graph.RegisterMakeSymbolFormatter(commonJSPackageUnitType, newSymbolFormatter)
}

func newSymbolFormatter(s *graph.Symbol) graph.SymbolFormatter {
	var si symbolData
	if len(s.Data) > 0 {
		if err := json.Unmarshal(s.Data, &si); err != nil {
			panic("unmarshal JavaScript symbol data: " + err.Error())
		}
	}
	return symbolFormatter{s, &si}
}

type symbolFormatter struct {
	symbol *graph.Symbol
	data   *symbolData
}

func (f symbolFormatter) Language() string { return "JavaScript" }

func (f symbolFormatter) DefKeyword() string {
	switch f.data.Kind {
	case Func:
		return "function"
	case Var:
		return "var"
	}
	return ""
}

func (f symbolFormatter) Kind() string { return f.data.Kind }

func (f symbolFormatter) Name(qual graph.Qualification) string {
	if f.data.Key.Namespace == "global" || f.data.Key.Namespace == "file" {
		return scopePathComponentsAfterAtSign(f.data.Key.Path)
	}
	switch qual {
	case graph.Unqualified:
		return f.symbol.Name
	case graph.ScopeQualified:
		return f.data.Key.Path
	case graph.DepQualified:
		return strings.TrimSuffix(filepath.Base(f.data.Key.Module), ".js") + "." + f.Name(graph.ScopeQualified)
	case graph.RepositoryWideQualified:
		return filepath.Join(f.symbol.Unit, strings.TrimSuffix(f.data.Key.Module, ".js")) + "." + f.Name(graph.ScopeQualified)
	case graph.LanguageWideQualified:
		return string(f.symbol.Repo) + "/" + f.Name(graph.RepositoryWideQualified)
	}
	panic("Name: unhandled qual " + string(qual))
}

func (f symbolFormatter) NameAndTypeSeparator() string {
	if f.data.IsFunc {
		return ""
	}
	return " "
}

func (f symbolFormatter) Type(qual graph.Qualification) string {
	var ts string
	if f.data.IsFunc {
		ts = strings.Replace(strings.TrimPrefix(f.data.Type, "fn"), ") -> ", ") ", -1)
	} else {
		ts = f.data.Type
	}

	ts = strings.Replace(ts, ": ?", "", -1)
	return ts
}
