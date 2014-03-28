package gog

import (
	"fmt"
	"go/ast"
	"go/token"
	"log"
	"path/filepath"
	"strings"

	"code.google.com/p/go.tools/go/loader"
	"code.google.com/p/go.tools/go/types"
)

func (g *Grapher) buildScopeInfo(pkgInfo *loader.PackageInfo) {
	for node, scope := range pkgInfo.Scopes {
		g.scopeNodes[scope] = node
	}
}

func (g *Grapher) path(obj types.Object) (path []string) {
	if path, present := g.paths[obj]; present {
		return path
	}

	var scope *types.Scope
	pkgInfo, astPath, _ := g.program.PathEnclosingInterval(obj.Pos(), obj.Pos())
	if astPath != nil {
		for _, node := range astPath {
			if s, hasScope := pkgInfo.Scopes[node]; hasScope {
				scope = s
			}
		}
	}
	if scope == nil {
		scope = obj.Parent()
	}

	if scope == nil {
		panic("no scope for object " + obj.String())
	}

	prefix, hasPath := g.scopePaths[scope]
	if !hasPath {
		panic("no scope path for scope " + scope.String())
	}
	path = append([]string{}, prefix...)
	p := g.program.Fset.Position(obj.Pos())
	path = append(path, obj.Name()+uniqID(p))
	return path

	panic("no scope node for object " + obj.String())
}

func uniqID(p token.Position) string {
	return fmt.Sprintf("$%s%d", strippedFilename(p.Filename), p.Offset)
}

func (g *Grapher) scopePath(prefix []string, s *types.Scope) []string {
	if path, present := g.scopePaths[s]; present {
		return path
	}
	path := append(prefix, g.scopeLabel(s)...)
	g.scopePaths[s] = path
	return path
}

func (g *Grapher) scopeLabel(s *types.Scope) (path []string) {
	node, present := g.scopeNodes[s]
	if !present {
		panic("no node found for scope " + s.String())
	}

	switch n := node.(type) {
	case *ast.File:
		return []string{}

	case *ast.Package:
		return []string{}

	case *ast.FuncType:
		// get func name
		_, astPath, _ := g.program.PathEnclosingInterval(n.Pos(), n.End())
		if false {
			log.Printf("----------")
			for i, n := range astPath {
				log.Printf("%d. %T %+v", i, n, n)
			}
		}
		if f, ok := astPath[0].(*ast.FuncDecl); ok {
			var path []string
			if f.Recv != nil {
				path = []string{derefNode(f.Recv.List[0].Type).(*ast.Ident).Name}
			}
			path = append(path, f.Name.Name)
			return path
		}
	}

	// get this scope's index in parent
	p := s.Parent()
	var prefix []string
	if fs, ok := g.scopeNodes[p].(*ast.File); ok {
		// avoid file scope collisions by using file index as well
		filename := g.program.Fset.Position(fs.Name.Pos()).Filename
		prefix = []string{fmt.Sprintf("$%s", strippedFilename(filename))}
	}
	for i := 0; i < p.NumChildren(); i++ {
		if p.Child(i) == s {
			filename := g.program.Fset.Position(node.Pos()).Filename
			return append(prefix, fmt.Sprintf("$%s%d", strippedFilename(filename), i))
		}
	}

	panic("unreachable")
}

func strippedFilename(filename string) string {
	return strings.TrimSuffix(filepath.Base(filename), ".go")
}

func (g *Grapher) assignPathsInPackage(pkgInfo *loader.PackageInfo) {
	pkg := pkgInfo.Pkg
	g.assignPaths(pkg.Scope(), []string{}, true, true)
}

func (g *Grapher) assignPaths(s *types.Scope, prefix []string, exported, pkgscope bool) {
	g.scopePaths[s] = prefix

	for _, name := range s.Names() {
		e := s.Lookup(name)
		path := append(append([]string{}, prefix...), name)
		g.paths[e] = path

		thisExported := exported && ast.IsExported(name)
		g.exported[e] = thisExported
		g.pkgscope[e] = pkgscope

		if tn, ok := e.(*types.TypeName); ok {
			// methods
			named := tn.Type().(*types.Named)
			g.assignMethodPaths(named, path, thisExported, pkgscope)

			// struct fields
			typ := derefType(tn.Type().Underlying())
			if styp, ok := typ.(*types.Struct); ok {
				g.assignStructFieldPaths(styp, path, thisExported, pkgscope)
			}
		} else if v, ok := e.(*types.Var); ok {
			// struct fields if type is anonymous struct
			if styp, ok := derefType(v.Type()).(*types.Struct); ok {
				g.assignStructFieldPaths(styp, path, thisExported, pkgscope)
			}
		}
	}

	for i := 0; i < s.NumChildren(); i++ {
		c := s.Child(i)
		childPrefix := prefix

		if path := g.scopePath(prefix, c); path != nil {
			childPrefix = append([]string{}, path...)
			pkgscope, exported = false, false
		}

		g.assignPaths(c, childPrefix, exported, pkgscope)
	}
}

func (g *Grapher) assignMethodPaths(named *types.Named, prefix []string, exported, pkgscope bool) {
	for i := 0; i < named.NumMethods(); i++ {
		m := named.Method(i)
		path := append(append([]string{}, prefix...), m.Name())
		g.paths[m] = path

		thisExported := exported && ast.IsExported(m.Name())
		g.exported[m] = thisExported
		g.pkgscope[m] = pkgscope

		if s := m.Scope(); s != nil {
			g.assignPaths(s, path, false, false)
		}
	}

	if iface, ok := named.Underlying().(*types.Interface); ok {
		for i := 0; i < iface.NumExplicitMethods(); i++ {
			m := iface.Method(i)
			path := append(append([]string{}, prefix...), m.Name())
			g.paths[m] = path

			thisExported := exported && ast.IsExported(m.Name())
			g.exported[m] = thisExported
			g.pkgscope[m] = pkgscope

			if s := m.Scope(); s != nil {
				g.assignPaths(s, path, false, false)
			}
		}
	}
}

func (g *Grapher) assignStructFieldPaths(styp *types.Struct, prefix []string, exported, pkgscope bool) {
	for i := 0; i < styp.NumFields(); i++ {
		f := styp.Field(i)
		path := append(append([]string{}, prefix...), f.Name())
		g.paths[f] = path

		thisExported := exported && ast.IsExported(f.Name())
		g.exported[f] = thisExported
		g.pkgscope[f] = pkgscope

		// recurse to anonymous structs (named structs are assigned directly)
		// TODO(sqs): handle arrays of structs, etc.
		if styp, ok := derefType(f.Type()).(*types.Struct); ok {
			g.assignStructFieldPaths(styp, path, thisExported, pkgscope)
		}
	}
}
