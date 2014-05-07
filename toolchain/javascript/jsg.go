package javascript

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"sourcegraph.com/sourcegraph/srcgraph/graph"
	"sourcegraph.com/sourcegraph/srcgraph/repo"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
)

type jsgOutput struct {
	Symbols []*Symbol
	Refs    []*Ref
}

type Symbol struct {
	ID       string
	Key      DefPath
	Type     string
	Exported bool

	Recv []*RefTarget

	File      string
	IdentSpan string `json:"ident"`
	DefnSpan  string `json:"defn"`

	Doc string

	Data *struct {
		NodeJS *struct {
			ModuleExports bool
		}
		AMD *struct {
			Module bool
		}
	}
}

func makeSymbolSpecificPath(sym *Symbol) string {
	if sym.Key.Namespace == "global" || sym.Key.Namespace == "file" {
		return scopePathComponentsAfterAtSign(sym.Key.Path)
	}
	return strings.TrimSuffix(filepath.Base(sym.Key.Module), ".js") + "." + sym.Key.Path
}

type Ref struct {
	File   string
	Span   string
	Target RefTarget
}

type RefTarget struct {
	Abstract  bool
	Path      string
	Origin    string
	Module    string
	Namespace string

	NodeJSCoreModule string

	NPMPackage *struct {
		Name            string
		Dir             string
		PackageJSONFile string
		Repository      *struct{ Type, URL string }
	}
}

var ErrSkipResolve = errors.New("skip resolution of this ref target")

func (t RefTarget) Resolve() (repo.URI, graph.SymbolPath, error) {
	// TODO(sqs): assume vcs type can be determined from the repoURL
	var uri repo.URI
	repoURL, _, err := t.Repository()
	if err != nil {
		if t.Origin == "ecma5" || t.Origin == "browser" {
			return "", "", ErrSkipResolve
		}
		return "", "", err
	}
	if repoURL != "" {
		uri = repo.MakeURI(repoURL)
	}

	dp, err := t.DefPath()
	if err != nil {
		return "", "", err
	}

	return uri, dp.symbolPath(), nil
}

var ErrNotAnNPMPackage = errors.New("not an npm package")

func (t RefTarget) ModuleRelativeToNPMPackage() (string, error) {
	if t.NPMPackage == nil {
		return "", ErrNotAnNPMPackage
	}
	return filepath.Rel(t.NPMPackage.Dir, t.Module)
}

var ErrUnknownTargetRepository = errors.New("couldn't determine target repository")

func (t *RefTarget) Repository() (url string, vcs repo.VCS, err error) {
	if t.NPMPackage != nil && t.NPMPackage.Repository != nil {
		return t.NPMPackage.Repository.URL, repo.VCS(t.NPMPackage.Repository.Type), nil
	}
	if t.Origin == "node" || t.NodeJSCoreModule != "" {
		return nodeStdlibRepoURL, repo.Git, nil
	}
	if !t.Abstract {
		// Current repository
		return "", "", nil
	}
	return "", "", ErrUnknownTargetRepository
}

func (t *RefTarget) DefPath() (*DefPath, error) {
	dp := &DefPath{Namespace: t.Namespace, Module: t.Module, Path: t.Path}

	if t.Origin == "node" && t.Namespace == "commonjs" {
		dp.Module = "lib/" + dp.Module + ".js"
	}
	if t.NodeJSCoreModule != "" {
		dp.Module = "lib/" + t.NodeJSCoreModule + ".js"
	}

	if t.NPMPackage != nil {
		var err error
		dp.Module, err = t.ModuleRelativeToNPMPackage()
		if err != nil {
			return nil, err
		}
	}
	return dp, nil
}

type DefPath struct {
	Namespace string
	Module    string
	Path      string
}

func (p DefPath) symbolPath() graph.SymbolPath {
	p.Path = strconv.QuoteToASCII(p.Path)
	p.Path = p.Path[1 : len(p.Path)-1]
	if p.Module == "" {
		return graph.SymbolPath(fmt.Sprintf("%s/-/%s", p.Namespace, p.Path))
	} else if p.Path == "" {
		return graph.SymbolPath(fmt.Sprintf("%s/%s", p.Namespace, p.Module))
	} else {
		return graph.SymbolPath(fmt.Sprintf("%s/%s/-/%s", p.Namespace, p.Module, strings.Replace(p.Path, ".", "/", -1)))
	}
}

func lastScopePathComponent(scopePath string) string {
	lastDot := strings.LastIndex(scopePath, ".")
	if lastDot == -1 {
		return scopePath
	}
	if lastDot == len(scopePath)-1 {
		return lastScopePathComponent(scopePath[:lastDot]) + "."
	}
	return scopePath[lastDot+1:]
}

func scopePathComponentsAfterAtSign(path string) string {
	parts := strings.Split(path, ".")
	if len(parts) == 1 {
		return path
	}
	for i := len(parts) - 2; i >= 0; i-- {
		part := parts[i]
		if strings.Contains(part, "@") {
			return strings.Join(parts[i+1:], ".")
		}
	}
	return path
}

func parseSpan(span string) (start, end int, err error) {
	sep := strings.Index(span, "-")
	if sep == -1 {
		return 0, 0, errors.New("no sep")
	}
	if sep == len(span)-1 {
		return 0, 0, errors.New("nothing after sep")
	}
	startstr, endstr := span[:sep], span[sep+1:]
	start, err = strconv.Atoi(startstr)
	if err != nil {
		return
	}
	end, err = strconv.Atoi(endstr)
	return
}

func convertSymbol(jsym *Symbol) (*graph.Symbol, []*graph.Ref, []*graph.Propagate, []*graph.Doc, error) {
	var refs []*graph.Ref
	var propgs []*graph.Propagate
	var docs []*graph.Doc

	// JavaScript symbol
	sym := &graph.Symbol{
		SymbolKey:    graph.SymbolKey{Path: jsym.Key.symbolPath()},
		Kind:         kind(jsym),
		SpecificKind: specificKind(jsym),
		Exported:     jsym.Exported,
		TypeExpr:     jsym.Type,
		Callable:     strings.HasPrefix(jsym.Type, "fn("),
	}

	if sym.SpecificKind == AMDModule || sym.SpecificKind == CommonJSModule {
		// File
		moduleFile := jsym.Key.Module
		moduleName := strings.TrimSuffix(jsym.Key.Module, ".js")
		sym.SpecificPath = strings.TrimSuffix(filepath.Base(moduleName), ".js")
		sym.Name = moduleName
		sym.File = moduleFile
		sym.DefStart = 0
		sym.DefEnd = 0 // TODO(sqs): get filesize
		sym.Exported = true
	} else {
		sym.Name = lastScopePathComponent(jsym.Key.Path)
		sym.SpecificPath = makeSymbolSpecificPath(jsym)
		sym.File = jsym.File

		if jsym.DefnSpan != "" {
			var err error
			sym.DefStart, sym.DefEnd, err = parseSpan(jsym.DefnSpan)
			if err != nil {
				return nil, nil, nil, nil, err
			}
		}
	}

	if jsym.Doc != "" {
		docs = append(docs, &graph.Doc{
			SymbolKey: sym.SymbolKey,
			Format:    "",
			Data:      strings.TrimPrefix(strings.TrimSpace(jsym.Doc), "* "),
		})
	}

	for _, recv := range jsym.Recv {
		srcRepo, srcPath, err := recv.Resolve()
		if err == ErrSkipResolve {
			continue
		}
		if err != nil {
			return nil, nil, nil, nil, err
		}
		propgs = append(propgs, &graph.Propagate{
			DstPath: sym.Path,
			SrcRepo: srcRepo,
			SrcPath: srcPath,
		})
	}

	return sym, refs, propgs, docs, nil
}

func convertRef(current unit.SourceUnit, jref *Ref) (*graph.Ref, error) {
	repoURI, path, err := jref.Target.Resolve()
	if err == ErrSkipResolve {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	start, end, err := parseSpan(jref.Span)
	if err != nil {
		return nil, err
	}

	ref := &graph.Ref{
		SymbolRepo:     repoURI,
		SymbolUnitType: unit.Type(current),
		SymbolUnit:     current.Name(),
		SymbolPath:     path,
		Def:            false, // TODO(sqs): determine if this is the Def
		File:           filepath.Join(jref.File),
		Start:          start,
		End:            end,
	}

	return ref, nil
}
