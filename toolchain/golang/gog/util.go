package gog

import (
	"bytes"
	"go/ast"
	"go/printer"
	"go/token"
	"sort"

	"code.google.com/p/go.tools/go/types"
)

func prettyPrint(n ast.Node) string {
	var b bytes.Buffer
	printer.Fprint(&b, token.NewFileSet(), n)
	s := b.String()
	if s == "" {
		return "(n/a)"
	} else {
		return s
	}
}

// Sort AST package files so that the result does not depend on map iteration order.
func sortedFiles(m map[string]*ast.File) []*ast.File {
	keylist := make([]string, len(m))
	i := 0
	for filename, _ := range m {
		keylist[i] = filename
		i++
	}
	sort.Strings(keylist)

	vallist := make([]*ast.File, len(m))
	for i, filename := range keylist {
		vallist[i] = m[filename]
	}
	return vallist
}

func makeSpan(fset *token.FileSet, node ast.Node) [2]int {
	pos := node.Pos()
	start := fset.Position(pos)
	return [2]int{start.Offset, start.Offset + int((node.End() - pos))}
}

func derefNode(n ast.Expr) ast.Expr {
	if n, ok := n.(*ast.StarExpr); ok {
		return n.X
	}
	return n
}

func derefType(t types.Type) types.Type {
	if pt, ok := t.(*types.Pointer); ok {
		return pt.Elem()
	}
	return t
}

func methodRecvTypeName(recvType ast.Expr) string {
	recvType = derefNode(recvType)
	return recvType.(*ast.Ident).Name
}
