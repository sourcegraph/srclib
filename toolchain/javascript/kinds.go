package javascript

import (
	"strings"

	"sourcegraph.com/sourcegraph/srcgraph/graph"
)

const (
	NPMPackage      = "npm_package"
	CommonJSModule  = "commonjs_module"
	AMDModule       = "amd_module"
	Func            = "func"
	ConstructorFunc = "constructor_func"
	Var             = "var"
	Property        = "property"
	Prototype       = "prototype"
)

func kind(s *Symbol) graph.SymbolKind {
	sk := specificKind(s)
	switch sk {
	case Property:
		return graph.Field
	case Prototype:
		return graph.Type
	case CommonJSModule, AMDModule:
		return graph.Module
	case NPMPackage:
		return graph.Package
	}
	return graph.SymbolKind(sk)
}

func specificKind(s *Symbol) string {
	if s.Data != nil {
		if s.Data.NodeJS != nil && s.Data.NodeJS.ModuleExports {
			return CommonJSModule
		}
		if s.Data.AMD != nil && s.Data.AMD.Module {
			return AMDModule
		}
	}

	if strings.HasSuffix(s.Key.Path, ".prototype") {
		return Prototype
	}

	if strings.HasPrefix(s.Type, "fn(") {
		return Func
	}

	if len(strings.Split(s.Key.Path, ".")) > 2 {
		return Property
	}

	return Var
}
