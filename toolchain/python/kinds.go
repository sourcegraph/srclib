package python

import "sourcegraph.com/sourcegraph/graph"

const (
	Package = "package"
	Module  = "module"

	// Other symbol kinds are defined in the Python code and passed through
	// verbatim (except for being lowercased): ATTRIBUTE, CLASS, CONSTRUCTOR,
	// etc.
)

var callablePythonSymbolKinds = map[string]bool{
	"CONSTRUCTOR": true,
	"FUNCTION":    true,
	"METHOD":      true,
}

var py2sgSymKindMap = map[string]graph.SymbolKind{
	"attribute":   graph.Field,
	"class":       graph.Type,
	"constructor": graph.Func,
	"function":    graph.Func,
	"method":      graph.Func,
	"module":      graph.Module,
	"package":     graph.Package,
	"parameter":   graph.Var,
	"scope":       graph.Var,
	"variable":    graph.Var,
}

var py2SpecificSymKindMap = map[string]string{
	"attribute":   "attr",
	"class":       "class",
	"constructor": "constructor",
	"function":    "func",
	"method":      "method",
	"module":      "module",
	"package":     "package",
	"parameter":   "param",
	"scope":       "var",
	"variable":    "var",
}
