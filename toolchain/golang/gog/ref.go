package gog

import (
	"go/ast"

	"code.google.com/p/go.tools/go/types"
)

func (g *Grapher) NewRef(node ast.Node, obj types.Object) (*Ref, error) {
	key, err := g.symbolKey(obj)
	if err != nil {
		return nil, err
	}

	pos := g.program.Fset.Position(node.Pos())
	return &Ref{
		File:   pos.Filename,
		Span:   makeSpan(g.program.Fset, node),
		Symbol: key,
	}, nil
}

type Ref struct {
	File   string
	Span   [2]int
	Symbol *SymbolKey

	// Def is true if ref is to the definition of Symbol, and false if it's to a
	// use of Symbol.
	Def bool
}
