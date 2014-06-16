package javascript

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/container"
	"sourcegraph.com/sourcegraph/srcgraph/graph"
	"sourcegraph.com/sourcegraph/srcgraph/grapher2"
	"sourcegraph.com/sourcegraph/srcgraph/repo"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
)

func init() {
	config.Register("jsg", &JSGConfig{})
	grapher2.Register(&CommonJSPackage{}, grapher2.DockerGrapher{defaultJSG})
}

// JSGConfig is custom configuration for node.js projects.
type JSGConfig struct {
	// Plugins is a map of plugin name to plugin configuration, for defining
	// plugins that should be loaded in jsg. jsg uses tern's plugin loading
	// system; see the jsg documentation for more information.
	Plugins map[string]interface{}
}

type jsg struct{ nodeVersion }

var defaultJSG = &jsg{defaultNode}

const (
	containerNodeCoreModulesDir = "/tmp/node_core_modules"
	jsgDir                      = "/usr/local/lib/node_modules/jsg"
)

func (v jsg) jsgConfig(c *config.Repository) *JSGConfig {
	jsgConfig, _ := c.Global["jsg"].(*JSGConfig)
	if jsgConfig == nil {
		jsgConfig = &JSGConfig{}
	}
	if jsgConfig.Plugins == nil {
		jsgConfig.Plugins = map[string]interface{}{}
	}
	if _, present := jsgConfig.Plugins["node"]; !present {
		// By default, use the node_core_modules dir that ships with jsg (for resolving refs to the node core).
		jsgConfig.Plugins["node"] = map[string]string{"coreModulesDir": containerNodeCoreModulesDir}
	}

	// Replace $(JSG_DIR) with the actual dir to jsg on the container, so that
	// configs can use jsg plugins defined in jsg's dependencies without having
	// to hardcode the dir on the container.
	const jsgDirVar = "$(JSG_DIR)"
	for name, v := range jsgConfig.Plugins {
		if strings.Contains(name, jsgDirVar) {
			delete(jsgConfig.Plugins, name)
			jsgConfig.Plugins[strings.Replace(name, jsgDirVar, jsgDir, -1)] = v
		}
	}

	return jsgConfig
}

func (v jsg) BuildGrapher(dir string, u unit.SourceUnit, c *config.Repository) (*container.Command, error) {
	pkg := u.(*CommonJSPackage)
	jsgConfig := v.jsgConfig(c)

	if len(pkg.sourceFiles()) == 0 {
		// No source files found for source unit; proceed without running grapher.
		return nil, nil
	}

	dockerfile, err := v.baseDockerfile()
	if err != nil {
		return nil, err
	}

	const (
		jsgVersion = "jsg@0.0.3"
		jsgGit     = "git://github.com/sourcegraph/jsg.git"
		jsgSrc     = jsgVersion
	)
	dockerfile = append(dockerfile, []byte("\n\nRUN npm install --quiet -g "+jsgSrc+"\n")...)

	// Copy the node core modules to the container.
	dockerfile = append(dockerfile, []byte("\nRUN cp -R "+jsgDir+"/testdata/node_core_modules "+containerNodeCoreModulesDir+"\n")...)

	jsgCmd, err := jsgCommand(jsgConfig.Plugins, nil, nil, pkg.sourceFiles())
	if err != nil {
		return nil, err
	}

	// Track test files so we can set the Test field on symbols efficiently.
	isTestFile := make(map[string]struct{}, len(pkg.TestFiles))
	for _, f := range pkg.TestFiles {
		isTestFile[f] = struct{}{}
	}

	containerDir := containerDir(dir)

	var preCmd []byte
	if pkg.PackageJSONFile != "" {
		// If there's a package.json file, `npm install` first.
		preCmd = []byte("WORKDIR " + containerDir + "\n" + fixPhantomJSHack + "\nRUN npm install --quiet --ignore-scripts --no-bin-links")
	}

	cmd := container.Command{
		Container: container.Container{
			Dockerfile:       dockerfile,
			AddDirs:          [][2]string{{dir, containerDir}},
			PreCmdDockerfile: preCmd,
			Cmd:              jsgCmd,
			Dir:              containerDir,
		},
		Transform: func(in []byte) ([]byte, error) {
			var o jsgOutput
			err := json.Unmarshal(in, &o)
			if err != nil {
				return nil, err
			}

			var o2 grapher2.Output

			for _, js := range o.Symbols {
				sym, refs, propgs, docs, err := convertSymbol(js)
				if err != nil {
					return nil, err
				}
				if sym == nil {
					continue
				}

				if _, isTest := isTestFile[sym.File]; isTest {
					sym.Test = true
				}

				o2.Symbols = append(o2.Symbols, sym)
				o2.Refs = append(o2.Refs, refs...)
				// TODO(sqs): handle propgs
				_ = propgs
				o2.Docs = append(o2.Docs, docs...)
			}
			for _, jr := range o.Refs {
				ref, err := convertRef(u, jr)
				if err != nil {
					return nil, err
				}
				if ref != nil {
					o2.Refs = append(o2.Refs, ref)
				}
			}

			return json.Marshal(o2)
		},
	}

	return &cmd, nil
}

func jsgCommand(plugins map[string]interface{}, defs []string, flags []string, origins []string) ([]string, error) {
	args := []string{"nodejs", "/usr/local/bin/jsg"}

	for name, config := range plugins {
		args = append(args, "--plugin")
		if config == nil {
			args = append(args, name)
		} else {
			configJSON, err := json.Marshal(config)
			if err != nil {
				return nil, err
			}
			args = append(args, name+"="+string(configJSON))
		}
	}

	for _, name := range defs {
		args = append(args, "--def", name)
	}

	args = append(args, flags...)
	args = append(args, origins...)

	return args, nil
}

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

	Data *jsgSymbolData
}

// jsgSymbolData is the "data" field output by jsg.
type jsgSymbolData struct {
	NodeJS *struct {
		ModuleExports bool
	}
	AMD *struct {
		Module bool
	}
}

// symbolData is stored in the graph.Symbol's Data field as JSON.
type symbolData struct {
	Kind string
	Key  DefPath
	*jsgSymbolData
	Type   string
	IsFunc bool
}

type Ref struct {
	File   string
	Span   string
	Target RefTarget
	Def    bool
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

func (t RefTarget) Resolve() (r repo.URI, unit string, path graph.SymbolPath, err error) {
	// TODO(sqs): assume vcs type can be determined from the repoURL
	var uri repo.URI
	repoURL, unit, _, err := t.Repository()
	if err != nil {
		if t.Origin == "ecma5" || t.Origin == "browser" {
			return "", "", "", ErrSkipResolve
		}
		return "", "", "", err
	}
	if repoURL != "" {
		uri = repo.MakeURI(repoURL)
	}

	dp, err := t.DefPath()
	if err != nil {
		return "", "", "", err
	}

	return uri, unit, dp.symbolPath(), nil
}

var ErrNotAnNPMPackage = errors.New("not an npm package")

func (t RefTarget) ModuleRelativeToNPMPackage() (string, error) {
	if t.NPMPackage == nil {
		return "", ErrNotAnNPMPackage
	}
	return filepath.Rel(t.NPMPackage.Dir, t.Module)
}

var ErrUnknownTargetRepository = errors.New("couldn't determine target repository")

func (t *RefTarget) Repository() (url string, unit string, vcs repo.VCS, err error) {
	if t.NPMPackage != nil && t.NPMPackage.Repository != nil {
		return t.NPMPackage.Repository.URL, t.NPMPackage.Name, repo.VCS(t.NPMPackage.Repository.Type), nil
	}
	if t.Origin == "node" || t.NodeJSCoreModule != "" {
		return nodeStdlibRepoURL, NodeJSStdlibUnit, repo.Git, nil
	}
	if !t.Abstract {
		// Current repository
		return "", "", "", nil
	}
	return "", "", "", ErrUnknownTargetRepository
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

func (p DefPath) symbolTreePath() graph.TreePath {
	// hack so we don't break on paths containing "//"
	p.Path = strings.Replace(p.Path, "//", "/", -1)
	p.Path = strconv.QuoteToASCII(p.Path)
	p.Path = p.Path[1 : len(p.Path)-1]

	namespaceComponents := strings.Split(p.Namespace, "/")
	for n := 0; n < len(namespaceComponents); n++ {
		namespaceComponents[n] = "-" + namespaceComponents[n]
	}
	ghostedNamespace := strings.Join(namespaceComponents, "/")

	if p.Module == "" {
		return graph.TreePath(fmt.Sprintf("%s/-/%s", ghostedNamespace, p.Path))
	} else if p.Path == "" {
		return graph.TreePath(fmt.Sprintf("%s/%s", ghostedNamespace, p.Module))
	} else {
		return graph.TreePath(fmt.Sprintf("%s/%s/-/%s", ghostedNamespace, p.Module, strings.Replace(p.Path, ".", "/", -1)))
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

	// unexported if it has (or is underneath) a name prefixed with "_" or that
	// contains "<i>" (the array element marker)
	exported := jsym.Exported && !strings.HasPrefix(jsym.Key.Path, "_") && !strings.Contains(jsym.Key.Path, "._") && !strings.Contains(jsym.Key.Path, "<i>")

	isFunc := strings.HasPrefix(jsym.Type, "fn(")
	path := jsym.Key.symbolPath()
	treePath := jsym.Key.symbolTreePath()
	if !treePath.IsValid() {
		return nil, nil, nil, nil, fmt.Errorf("'%s' is not a valid tree-path", treePath)
	}

	// JavaScript symbol
	sym := &graph.Symbol{
		SymbolKey: graph.SymbolKey{Path: path},
		TreePath:  treePath,
		Kind:      kind(jsym),
		Exported:  exported,
		Callable:  isFunc,
	}

	sd := symbolData{
		Kind:          jsKind(jsym),
		Key:           jsym.Key,
		jsgSymbolData: jsym.Data,
		Type:          jsym.Type,
		IsFunc:        isFunc,
	}

	if sd.Kind == AMDModule || sd.Kind == CommonJSModule {
		// File
		moduleFile := jsym.Key.Module
		moduleName := strings.TrimSuffix(jsym.Key.Module, ".js")
		sym.Name = moduleName
		sym.File = moduleFile
		sym.DefStart = 0
		sym.DefEnd = 0 // TODO(sqs): get filesize
		sym.Exported = true
	} else {
		sym.Name = lastScopePathComponent(jsym.Key.Path)
		sym.File = jsym.File

		if jsym.DefnSpan != "" {
			var err error
			sym.DefStart, sym.DefEnd, err = parseSpan(jsym.DefnSpan)
			if err != nil {
				return nil, nil, nil, nil, err
			}
		}
	}

	// HACK TODO(sqs): some avals have an origin in this project but a file
	// outside of it, and they refer to defs outside of this project. but
	// because the origin is in this project, they get emitted as symbols in
	// this project. fix this in jsg dump.js (most likely).
	if strings.Contains(string(sym.Path), "/node_core_modules/") {
		return nil, nil, nil, nil, nil
	}

	if jsym.Doc != "" {
		docs = append(docs, &graph.Doc{
			SymbolKey: sym.SymbolKey,
			Format:    "",
			Data:      strings.TrimPrefix(strings.TrimSpace(jsym.Doc), "* "),
		})
	}

	for _, recv := range jsym.Recv {
		srcRepo, srcUnit, srcPath, err := recv.Resolve()
		if err == ErrSkipResolve {
			continue
		}
		if err != nil {
			return nil, nil, nil, nil, err
		}
		propgs = append(propgs, &graph.Propagate{
			DstPath: sym.Path,
			SrcRepo: srcRepo,
			SrcUnit: srcUnit,
			SrcPath: srcPath,
		})
	}

	if b, err := json.Marshal(sd); err != nil {
		return nil, nil, nil, nil, err
	} else {
		sym.Data = b
	}

	return sym, refs, propgs, docs, nil
}

func convertRef(current unit.SourceUnit, jref *Ref) (*graph.Ref, error) {
	repoURI, u, path, err := jref.Target.Resolve()
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
		SymbolUnit:     u,
		SymbolPath:     path,
		Def:            jref.Def,
		File:           filepath.Join(jref.File),
		Start:          start,
		End:            end,
	}

	return ref, nil
}
