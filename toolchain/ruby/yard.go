package ruby

import (
	"path/filepath"
	"strings"

	"sourcegraph.com/sourcegraph/srcgraph/graph"
)

var StdlibGemNameSentinel = "<<RUBY_STDLIB>>"

type rubyObject struct {
	Name      string `json:"name"`
	File      string `json:"file"`
	Kind      string `json:"kind"`
	Module    string `json:"module"`
	DefStart  int    `json:"defStart"`
	DefEnd    int    `json:"defEnd"`
	Path      string `json:"path"`
	Exported  bool   `json:"exported"`
	Docstring string `json:"docstring"`

	// RubySonar fields (unused)
	Type     string `json:"type"`
	TypeExpr string `json:"type_expr"`
}

func (s *rubyObject) toSymbol() *graph.Symbol {
	_, notExported := rubyObjectTypeUnexported[s.Type]
	relFile, _ := filepath.Rel(srcRoot, s.File)
	return &graph.Symbol{
		SymbolKey:    graph.SymbolKey{Path: rubyPathToSymbolPath(s.Path)},
		SpecificPath: s.Path,
		Kind:         rubyObjectTypeMap[s.Type],
		SpecificKind: s.Type,
		Name:         s.Name,
		Exported:     !notExported,
		Callable:     s.Type == "method",
		File:         relFile,
		DefStart:     s.DefStart,
		DefEnd:       s.DefEnd,
		TypeExpr:     s.TypeExpr,
	}
}

var rubyObjectTypeMap = map[string]graph.SymbolKind{
	"method":           graph.Func,
	"constant":         graph.Const,
	"class":            graph.Type,
	"module":           graph.Module,
	"localvariable":    graph.Var,
	"instancevariable": graph.Var,
	"classvariable":    graph.Var,
}

// rubyObjectTypeUnexported is a map where membership indicates that Ruby
// objects of this type are NOT exported. Defined as the inverse because there
// are fewer unexported types than exported types.
var rubyObjectTypeUnexported = map[string]struct{}{
	"instancevariable": struct{}{},
	"classvariable":    struct{}{},
	"localvariable":    struct{}{},
}

type rubyRef struct {
	Target                 string `json:"target"`
	TargetOriginYardocFile string `json:"target_origin_yardoc_file"`
	Kind                   string `json:"kind"`
	File                   string `json:"file"`
	Start                  int    `json:"start"`
	End                    int    `json:"end"`
}

func (r *rubyRef) toRef() (ref *graph.Ref, targetOrigin string) {
	if r.Kind == "" {
		r.Kind = "ident"
	}
	relFile, _ := filepath.Rel(srcRoot, r.File)
	return &graph.Ref{
		SymbolPath: rubyPathToSymbolPath(r.Target),
		File:       relFile,
		Start:      r.Start,
		End:        r.End,
	}, r.TargetOriginYardocFile
}

func rubyPathToSymbolPath(path string) graph.SymbolPath {
	p := strings.Replace(strings.Replace(strings.Replace(strings.Replace(path, ".rb", "_rb", -1), "::", "/", -1), "#", "/$methods/", -1), ".", "/$classmethods/", -1)
	return graph.SymbolPath(strings.TrimPrefix(p, "/"))
}
