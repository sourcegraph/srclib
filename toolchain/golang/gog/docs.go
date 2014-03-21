package gog

// import (
// 	"bytes"
// 	"code.google.com/p/go.tools/go/types"
// 	"github.com/sourcegraph/srcscan"
// 	"go/doc"
// 	"sourcegraph.com/sourcegraph/graph"
// )

// func (a *analyzer) graphDocs(unit srcscan.Unit) error {
// 	// Reparse AST here because go/doc modifies the AST during processing.
// 	astpkg, _, err := parsePkgAST(a.buildPkg, a.ctx.UnitAbsPath(unit))
// 	if err != nil {
// 		return err
// 	}

// 	docpkg := doc.New(astpkg, a.buildPkg.ImportPath, doc.AllDecls|doc.AllMethods)
// 	typkg := types.NewPackage(docpkg.ImportPath, docpkg.Name, nil)

// 	var emitDoc = func(path graph.SymbolPath, docstring string) {
// 		if docstring != "" {
// 			var buf bytes.Buffer
// 			doc.ToHTML(&buf, docstring, nil)
// 			a.ctx.Doc(&graph.Doc{SymbolKey: graph.SymbolKey{Path: path}, Body: buf.String()})
// 		}
// 	}
// 	var emitDocForValue = func(val *doc.Value) {
// 		for _, name := range val.Names {
// 			if name == "_" {
// 				continue
// 			}
// 			emitDoc(a.varSymbolPath(types.NewVar(0, typkg, name, nil), valSuffix), val.Doc)
// 		}
// 	}

// 	emitDoc(a.pkgSymbolPath(typkg), docpkg.Doc)
// 	for _, cnst := range docpkg.Consts {
// 		// Pretends this is a var because go/types currently has no NewConst func.
// 		// See https://code.google.com/p/go/issues/detail?id=5563.
// 		// TODO(sqs): make this use NewFunc if one is ever added
// 		emitDocForValue(cnst)
// 	}

// 	for _, vari := range docpkg.Vars {
// 		emitDocForValue(vari)
// 	}

// 	for _, fun := range docpkg.Funcs {
// 		// Pretends this is a var because go/types currently has no NewFunc func.
// 		// TODO(sqs): make this use NewFunc if one is ever added
// 		emitDoc(a.varSymbolPath(types.NewVar(0, typkg, fun.Name, nil), ""), fun.Doc)
// 	}

// 	for _, typ := range docpkg.Types {
// 		typobj := types.NewTypeName(0, typkg, typ.Name, nil)
// 		emitDoc(a.symbolPath(typobj), typ.Doc)
// 		for _, cnst := range typ.Consts {
// 			emitDocForValue(cnst)
// 		}
// 		for _, vari := range typ.Vars {
// 			emitDocForValue(vari)
// 		}
// 		for _, fun := range typ.Funcs {
// 			// TODO(sqs): make this use NewFunc when we upgrade go/types
// 			emitDoc(a.varSymbolPath(types.NewVar(0, typkg, fun.Name, nil), ""), fun.Doc)
// 		}
// 		for _, fun := range typ.Methods {
// 			// TODO(sqs): make this use NewFunc when we upgrade go/types
// 			emitDoc(a.symbolPath(typobj).WithJoinedPath(fun.Name), fun.Doc)
// 		}
// 	}

// 	return nil
// }
