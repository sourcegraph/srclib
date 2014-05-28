// +build off

package golang

import (
	"net/url"
	"path/filepath"

	"go/build"
	"go/token"
	"sort"

	"github.com/sourcegraph/go-vcsurl"
	"sourcegraph.com/sourcegraph/dep"
)

func (p *GoPackage) RawDependencies() ([]*dep.RawDependency, error) {
	deps := make([]*dep.RawDependency, 0)
	var addDeps = func(importPos map[string][]token.Position) {
		for imp, positions := range importPos {
			for _, pos := range positions {
				deps = append(deps, &dep.RawDependency{
					DefFile:  pos.Filename,
					DefStart: pos.Offset,
					DefEnd:   pos.Offset + len(imp) + 2, // length of import path plus surrounding quotes
					Target:   imp,
				})
			}
		}
	}
	pkg, err := p.buildPackage(nil)
	if err != nil {
		return nil, err
	}
	addDeps(pkg.ImportPos)
	addDeps(pkg.TestImportPos)
	addDeps(pkg.XTestImportPos)

	return deps, nil
}

func (p *GoPackage) buildPackage(c *build.Context) (*build.Package, error) {
	if c == nil {
		c = &build.Default
	}
	pkg, err := c.ImportDir(p.Dir, 0)
	if err != nil {
		return nil, err
	}
	if p.ImportPath != "" {
		pkg.ImportPath = p.ImportPath
	}
	return pkg, nil
}

// dependencyImportPaths returns a sorted list of all import paths (including
// test imports) directly imported by this package.
func (u *GoPackage) dependencyImportPaths(c *build.Context) ([]string, error) {
	pkg, err := u.BuildPackage(c)
	if err != nil {
		return nil, err
	}

	importPathMap := make(map[string]struct{}, len(pkg.Imports))
	var addDeps = func(importPaths []string) {
		for _, p := range importPaths {
			importPathMap[p] = struct{}{}
		}
	}
	addDeps(pkg.Imports)
	addDeps(pkg.TestImports)
	addDeps(pkg.XTestImports)

	importPaths := make([]string, len(importPathMap))
	i := 0
	for importPath, _ := range importPathMap {
		importPaths[i] = importPath
		i++
	}
	sort.Strings(importPaths)
	return importPaths, nil
}

func (t *goToolchain) ResolveDependencies(deps []*dep.RawDependency) (resolved map[*dep.RawDependency]*dep.ResolvedDependency, unresolvable map[*dep.RawDependency]error, err error) {
	// Group by target import path.
	depsByImportPath := make(map[string][]*dep.RawDependency)
	for _, d := range deps {
		importPath := d.Target.(string)
		depsByImportPath[importPath] = append(depsByImportPath[importPath], d)
	}

	var addResolvedDep = func(d dep.ResolvedDependency, fromRawDeps []*dep.RawDependency) {
		if resolved == nil {
			resolved = make(map[*dep.RawDependency]*dep.ResolvedDependency)
		}
		for _, fromRawDep := range fromRawDeps {
			resolved[fromRawDep] = &dep.ResolvedDependency{
				TargetRepositoryCloneURL: d.TargetRepositoryCloneURL,
				TargetRepositoryVCS:      d.TargetRepositoryVCS,
				TargetSourceUnit:         d.TargetSourceUnit,
				TargetSourceUnitType:     d.TargetSourceUnitType,
				TargetVersion:            d.TargetVersion,
				TargetRevision:           d.TargetRevision,
			}
		}
	}
	var addUnresolvableDep = func(fromRawDeps []*dep.RawDependency, err error) {
		if unresolvable == nil {
			unresolvable = make(map[*dep.RawDependency]error)
		}
		for _, fromRawDep := range fromRawDeps {
			unresolvable[fromRawDep] = err
		}
	}

	// Resolve each target import path.
	for importPath, deps := range depsByImportPath {
		var resolvedDep dep.ResolvedDependency
		if _, isBuiltin := t.Stdlib.BuiltinPackages[importPath]; isBuiltin {
			resolvedDep.TargetRepositoryCloneURL = t.Stdlib.RepositoryCloneURL
			resolvedDep.TargetRepositoryVCS = t.Stdlib.RepositoryVCS
			resolvedDep.TargetSourceUnit = importPath
			resolvedDep.TargetSourceUnitType = "GoPackage"
		} else {
			urlinfo, err := vcsurl.Parse(importPath)
			if err != nil {
				addUnresolvableDep(deps, err)
				continue
			}
			resolvedDep.TargetRepositoryCloneURL = urlinfo.CloneURL
			resolvedDep.TargetRepositoryVCS = urlinfo.VCS

			var u *url.URL
			u, err = url.Parse(urlinfo.Link())
			if err != nil {
				addUnresolvableDep(deps, err)
				continue
			}
			resolvedDep.TargetSourceUnit, _ = filepath.Rel(u.Host+u.Path, importPath)
			resolvedDep.TargetSourceUnitType = "GoPackage"
			resolvedDep.TargetRevision = "master" // TODO(sqs): find actual TargetRevision
		}
		addResolvedDep(resolvedDep, deps)
	}

	return
}
