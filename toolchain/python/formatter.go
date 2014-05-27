package python

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"sourcegraph.com/sourcegraph/srcgraph/graph"
)

func init() {
	graph.RegisterMakeSymbolFormatter(pythonUnitType, newSymbolFormatter)
}

func newSymbolFormatter(s *graph.Symbol) graph.SymbolFormatter {
	var si symbolData
	if err := json.Unmarshal(s.Data, &si); err != nil {
		panic("unmarshal Python symbol data: " + err.Error())
	}
	return symbolFormatter{s, &si}
}

type symbolFormatter struct {
	symbol *graph.Symbol
	data   *symbolData
}

func (f symbolFormatter) Language() string { return "Python" }

func (f symbolFormatter) DefKeyword() string {
	if f.isFunc() {
		return "def"
	}
	if f.data.Kind == "class" {
		return "class"
	}
	return ""
}

func (f symbolFormatter) Kind() string { return f.data.Kind }

func dotted(slashed string) string { return strings.Replace(slashed, "/", ".", -1) }

func (f symbolFormatter) Name(qual graph.Qualification) string {
	if qual == graph.Unqualified {
		return f.symbol.Name
	}

	module := strings.TrimSuffix(f.symbol.File, ".py")
	relPath := strings.TrimPrefix(strings.TrimPrefix(string(f.symbol.Path), module), "/")

	switch qual {
	case graph.ScopeQualified:
		return dotted(relPath)
	case graph.DepQualified:
		return dotted(filepath.Join(filepath.Base(module), relPath))
	case graph.RepositoryWideQualified:
		return dotted(filepath.Join(module, relPath))
	case graph.LanguageWideQualified:
		return string(f.symbol.Repo) + "/" + f.Name(graph.RepositoryWideQualified)
	}
	panic("Name: unhandled qual " + string(qual))
}

func (f symbolFormatter) isFunc() bool {
	k := f.data.Kind
	return k == "function" || k == "method" || k == "constructor"
}

func (f symbolFormatter) NameAndTypeSeparator() string {
	if f.isFunc() {
		return ""
	}
	return " "
}

func (f symbolFormatter) Type(qual graph.Qualification) string {
	return f.data.FuncSignature
}
