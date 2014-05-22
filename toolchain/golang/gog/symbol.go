package gog

import (
	"fmt"
	"go/ast"
	"path/filepath"
	"strings"

	"code.google.com/p/go.tools/go/loader"
	"code.google.com/p/go.tools/go/types"
)

type SymbolKey struct {
	PackageImportPath string
	Path              []string
}

func (s *SymbolKey) String() string {
	return s.PackageImportPath + "#" + strings.Join(s.Path, ".")
}

type Symbol struct {
	Name string

	*SymbolKey

	File      string
	IdentSpan [2]int
	DeclSpan  [2]int

	SymbolInfo
}

type SymbolInfo struct {
	// Exported is whether this symbol is exported.
	Exported bool

	// PkgScope is whether this symbol is in Go package scope.
	PkgScope bool

	// TypeString is a string describing this symbol's Go type.
	TypeString string

	// UnderlyingTypeString is the function or method signature, if this is a function or method.
	UnderlyingTypeString string `json:",omitempty"`

	// Kind is the kind of Go thing this symbol is: struct, interface, func,
	// package, etc.
	Kind string `json:",omitempty"`
}

// NewSymbol creates a new Symbol.
func (g *Grapher) NewSymbol(obj types.Object, declIdent *ast.Ident) (*Symbol, error) {
	// Find the AST node that declares this symbol.
	var declNode ast.Node
	_, astPath, _ := g.program.PathEnclosingInterval(declIdent.Pos(), declIdent.End())
	for _, node := range astPath {
		switch node.(type) {
		case *ast.FuncDecl, *ast.GenDecl, *ast.ValueSpec, *ast.TypeSpec, *ast.Field, *ast.DeclStmt, *ast.AssignStmt:
			declNode = node
			goto found
		}
	}
found:
	if declNode == nil {
		return nil, fmt.Errorf("On ident %s at %s: no DeclNode found (using PathEnclosingInterval)", declIdent.Name, g.program.Fset.Position(declIdent.Pos()))
	}

	key, info, err := g.symbolInfo(obj)
	if err != nil {
		return nil, err
	}

	si := SymbolInfo{
		Exported: info.exported,
		PkgScope: info.pkgscope,
		Kind:     symbolKind(obj),
	}

	if typ := obj.Type(); typ != nil {
		si.TypeString = typ.String()
		if utyp := typ.Underlying(); utyp != nil {
			si.UnderlyingTypeString = utyp.String()
		}
	}

	return &Symbol{
		Name: obj.Name(),

		SymbolKey: key,

		File:      g.program.Fset.Position(declIdent.Pos()).Filename,
		IdentSpan: makeSpan(g.program.Fset, declIdent),
		DeclSpan:  makeSpan(g.program.Fset, declNode),

		SymbolInfo: si,
	}, nil
}

// NewPackageSymbol creates a new Symbol that represents a Go package.
func (g *Grapher) NewPackageSymbol(pkgInfo *loader.PackageInfo, pkg *types.Package) (*Symbol, error) {
	var pkgDir string
	if len(pkgInfo.Files) > 0 {
		pkgDir = filepath.Dir(g.program.Fset.Position(pkgInfo.Files[0].Package).Filename)
	}

	return &Symbol{
		Name: pkg.Name(),

		SymbolKey: &SymbolKey{PackageImportPath: pkg.Path(), Path: []string{}},

		File: pkgDir,

		SymbolInfo: SymbolInfo{
			Exported: true,
			Kind:     Package,
		},
	}, nil
}

func symbolKind(obj types.Object) string {
	switch obj := obj.(type) {
	case *types.PkgName:
		return Package
	case *types.Const:
		return Const
	case *types.TypeName:
		return Type
	case *types.Var:
		return Var
	case *types.Func:
		sig := obj.Type().(*types.Signature)
		if sig.Recv() == nil {
			return Func
		} else {
			return Method
		}
	default:
		panic(fmt.Sprintf("unhandled obj type %T", obj))
	}
}
