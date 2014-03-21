package gog

import (
	"go/ast"
	"sort"
	"sync"

	"code.google.com/p/go.tools/go/exact"
	_ "code.google.com/p/go.tools/go/gcimporter"
	"code.google.com/p/go.tools/go/loader"
	"code.google.com/p/go.tools/go/types"
)

type Output struct {
	Symbols []*Symbol
	Refs    []*Ref
}

type Grapher struct {
	program *loader.Program

	// imported is the set of imported packages' import paths (that we should emit symbols
	// from).
	imported map[string]struct{}

	symbolCacheLock sync.Mutex
	symbolInfoCache map[types.Object]*symbolInfo
	symbolKeyCache  map[types.Object]*SymbolKey

	structFields map[*types.Var]*structField

	scopeNodes map[*types.Scope]ast.Node

	paths      map[types.Object][]string
	scopePaths map[*types.Scope][]string
	exported   map[types.Object]bool
	pkgscope   map[types.Object]bool

	Output

	// skipResolve is the set of *ast.Idents that the grapher encountered but
	// did not resolve (by design). Idents in this set are omitted from the list
	// of unresolved idents in the tests.
	skipResolve map[*ast.Ident]struct{}
}

func New(prog *loader.Program) *Grapher {
	imported := make(map[string]struct{})
	for importPath, _ := range prog.Imported {
		imported[importPath] = struct{}{}
	}

	g := &Grapher{
		program:         prog,
		imported:        imported,
		symbolInfoCache: make(map[types.Object]*symbolInfo),
		symbolKeyCache:  make(map[types.Object]*SymbolKey),

		structFields: make(map[*types.Var]*structField),

		scopeNodes: make(map[*types.Scope]ast.Node),

		paths:      make(map[types.Object][]string),
		scopePaths: make(map[*types.Scope][]string),
		exported:   make(map[types.Object]bool),
		pkgscope:   make(map[types.Object]bool),

		skipResolve: make(map[*ast.Ident]struct{}),
	}

	for _, pkgInfo := range sortedPkgs(prog.AllPackages) {
		g.buildStructFields(pkgInfo)
		g.buildScopeInfo(pkgInfo)
		g.assignPathsInPackage(pkgInfo)
	}

	return g
}

func sortedPkgs(m map[*types.Package]*loader.PackageInfo) []*loader.PackageInfo {
	var pis []*loader.PackageInfo
	for _, pi := range m {
		pis = append(pis, pi)
	}
	sort.Sort(packageInfos(pis))
	return pis
}

type packageInfos []*loader.PackageInfo

func (pi packageInfos) Len() int           { return len(pi) }
func (pi packageInfos) Less(i, j int) bool { return pi[i].Pkg.Path() < pi[j].Pkg.Path() }
func (pi packageInfos) Swap(i, j int)      { pi[i], pi[j] = pi[j], pi[i] }

func (g *Grapher) addSymbol(symbol *Symbol) {
	//	log.Printf("SYM %v %v", symbol.SymbolKey.PackageImportPath, symbol.SymbolKey.Path)
	g.Symbols = append(g.Symbols, symbol)
}

func (g *Grapher) addRef(ref *Ref) {
	//	log.Printf("REF %v %v at %s:%v", ref.Symbol.PackageImportPath, ref.Symbol.Path, ref.File, ref.Span)
	g.Refs = append(g.Refs, ref)
}

func (g *Grapher) GraphImported() error {
	for _, pkgInfo := range g.program.Imported {
		err := g.Graph(pkgInfo)
		if err != nil {
			return err
		}
	}
	return nil
}

func (g *Grapher) GraphAll() error {
	for _, pkgInfo := range g.program.AllPackages {
		err := g.Graph(pkgInfo)
		if err != nil {
			return err
		}
	}
	return nil
}

func (g *Grapher) Graph(pkgInfo *loader.PackageInfo) error {
	seen := make(map[ast.Node]struct{})
	skipResolveObjs := make(map[types.Object]struct{})

	for selExpr, sel := range pkgInfo.Selections {
		switch sel.Kind() {
		case types.PackageObj:
			pkg := sel.Obj().Pkg()
			pkgIdent := selExpr.X.(*ast.Ident)
			pkgObj := types.NewPkgName(selExpr.X.Pos(), pkg, pkg.Name())
			ref, err := g.NewRef(pkgIdent, pkgObj)
			if err != nil {
				return err
			}
			g.addRef(ref)

			seen[pkgIdent] = struct{}{}
		}
	}

	for node, obj := range pkgInfo.Implicits {
		if importSpec, ok := node.(*ast.ImportSpec); ok {
			ref, err := g.NewRef(importSpec, obj)
			if err != nil {
				return err
			}
			g.addRef(ref)
			seen[importSpec] = struct{}{}
		} else if x, ok := node.(*ast.Ident); ok {
			g.skipResolve[x] = struct{}{}
		} else if _, ok := node.(*ast.CaseClause); ok {
			// type-specific *Var for each type switch case clause
			skipResolveObjs[obj] = struct{}{}
		}
	}

	pkgSym, err := g.NewPackageSymbol(pkgInfo, pkgInfo.Pkg)
	if err != nil {
		return err
	}
	g.addSymbol(pkgSym)

	for ident, obj := range pkgInfo.Defs {
		_, isLabel := obj.(*types.Label)
		if obj == nil || ident.Name == "_" || isLabel {
			g.skipResolve[ident] = struct{}{}
			continue
		}

		if v, isVar := obj.(*types.Var); isVar && obj.Pos() != ident.Pos() && !v.IsField() {
			// If this is an assign statement reassignment of existing var, treat this as a
			// use (not a def).
			pkgInfo.Uses[ident] = obj
			continue
		}

		// don't treat import aliases as things that belong to this package
		_, isPkg := obj.(*types.PkgName)

		if !isPkg {
			sym, err := g.NewSymbol(obj, ident)
			if err != nil {
				return err
			}
			g.addSymbol(sym)
		}

		ref, err := g.NewRef(ident, obj)
		if err != nil {
			return err
		}
		ref.Def = true
		g.addRef(ref)
	}

	for ident, obj := range pkgInfo.Uses {
		if _, isLabel := obj.(*types.Label); isLabel {
			g.skipResolve[ident] = struct{}{}
			continue
		}

		if obj == nil || ident == nil || ident.Name == "_" {
			continue
		}

		if _, skip := skipResolveObjs[obj]; skip {
			g.skipResolve[ident] = struct{}{}
		}

		if _, seen := seen[ident]; seen {
			continue
		}

		if _, isLabel := obj.(*types.Label); isLabel {
			continue
		}

		ref, err := g.NewRef(ident, obj)
		if err != nil {
			return err
		}
		g.addRef(ref)
	}

	// Find refs to current package in the "package" clause in each file.
	for _, f := range pkgInfo.Files {
		pkgObj := types.NewPkgName(f.Name.Pos(), pkgInfo.Pkg, pkgInfo.Pkg.Name())
		ref, err := g.NewRef(f.Name, pkgObj)
		if err != nil {
			return err
		}
		g.addRef(ref)
	}

	return nil
}

type symbolInfo struct {
	exported bool
	pkgscope bool
}

func (g *Grapher) symbolKey(obj types.Object) (*SymbolKey, error) {
	key, _, err := g.symbolInfo(obj)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func (g *Grapher) symbolInfo(obj types.Object) (*SymbolKey, *symbolInfo, error) {
	key, info := g.lookupSymbolInfo(obj)
	if key != nil && info != nil {
		return key, info, nil
	}

	// Don't block while we traverse the AST to construct the object path. We
	// might duplicate effort, but it's better than allowing only one goroutine
	// to do this at a time.

	key, info, err := g.makeSymbolInfo(obj)
	if err != nil {
		return nil, nil, err
	}

	g.symbolCacheLock.Lock()
	defer g.symbolCacheLock.Unlock()
	g.symbolKeyCache[obj] = key
	g.symbolInfoCache[obj] = info
	return key, info, nil
}

func (g *Grapher) lookupSymbolInfo(obj types.Object) (*SymbolKey, *symbolInfo) {
	g.symbolCacheLock.Lock()
	defer g.symbolCacheLock.Unlock()
	return g.symbolKeyCache[obj], g.symbolInfoCache[obj]
}

func (g *Grapher) makeSymbolInfo(obj types.Object) (*SymbolKey, *symbolInfo, error) {
	switch obj := obj.(type) {
	case *types.Builtin:
		return &SymbolKey{"builtin", []string{obj.Name()}}, &symbolInfo{pkgscope: false, exported: true}, nil
	case *types.Nil:
		return &SymbolKey{"builtin", []string{"nil"}}, &symbolInfo{pkgscope: false, exported: true}, nil
	case *types.TypeName:
		if basic, ok := obj.Type().(*types.Basic); ok {
			return &SymbolKey{"builtin", []string{basic.Name()}}, &symbolInfo{pkgscope: false, exported: true}, nil
		}
		if obj.Name() == "error" {
			return &SymbolKey{"builtin", []string{obj.Name()}}, &symbolInfo{pkgscope: false, exported: true}, nil
		}
	case *types.PkgName:
		return &SymbolKey{obj.Pkg().Path(), []string{}}, &symbolInfo{pkgscope: false, exported: true}, nil
	case *types.Const:
		var pkg string
		if obj.Pkg() == nil {
			pkg = "builtin"
		} else {
			pkg = obj.Pkg().Path()
		}
		if obj.Val().Kind() == exact.Bool {
			return &SymbolKey{pkg, []string{obj.Name()}}, &symbolInfo{pkgscope: false, exported: true}, nil
		}
	}

	if obj.Pkg() == nil {
		// builtin
		return &SymbolKey{"builtin", []string{obj.Name()}}, &symbolInfo{pkgscope: false, exported: true}, nil
	}

	return &SymbolKey{obj.Pkg().Path(), g.path(obj)}, &symbolInfo{pkgscope: g.pkgscope[obj], exported: g.exported[obj]}, nil
}
