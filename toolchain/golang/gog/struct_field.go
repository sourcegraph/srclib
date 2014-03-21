package gog

import (
	"code.google.com/p/go.tools/go/loader"
	"code.google.com/p/go.tools/go/types"
)

type structField struct {
	*types.Var
	parent types.Type
}

func (g *Grapher) buildStructFields(pkgInfo *loader.PackageInfo) {
	for _, obj := range pkgInfo.Defs {
		if tn, ok := obj.(*types.TypeName); ok {
			typ := tn.Type().Underlying()
			if st, ok := typ.(*types.Struct); ok {
				for i := 0; i < st.NumFields(); i++ {
					sf := st.Field(i)
					g.structFields[sf] = &structField{sf, tn.Type()}
				}
			}
		}
	}

	for selExpr, sel := range pkgInfo.Selections {
		switch sel.Kind() {
		case types.FieldVal:
			rt := derefType(sel.Recv())
			var pkg *types.Package
			switch rt := rt.(type) {
			case *types.Named:
				pkg = rt.Obj().Pkg()
			case *types.Struct:
				pkg = sel.Obj().Pkg()
			default:
				panic("unhandled field recv type " + rt.String())
			}
			sfobj, _, _ := types.LookupFieldOrMethod(derefType(sel.Recv()), pkg, selExpr.Sel.Name)

			// Record that this field is in this struct so we can construct the
			// right symbol path to the field.
			sf, _ := sfobj.(*types.Var)
			g.structFields[sf] = &structField{sf, rt}
		}
	}
}
