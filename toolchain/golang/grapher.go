package golang

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"strings"

	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/container"
	"sourcegraph.com/sourcegraph/srcgraph/graph"
	"sourcegraph.com/sourcegraph/srcgraph/grapher2"
	"sourcegraph.com/sourcegraph/srcgraph/repo"
	"sourcegraph.com/sourcegraph/srcgraph/toolchain/golang/gog"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
)

func init() {
	grapher2.Register(&Package{}, grapher2.DockerGrapher{defaultGoVersion})
}

func (v *goVersion) BuildGrapher(dir string, unit unit.SourceUnit, c *config.Repository) (*container.Command, error) {
	gogBinPath := filepath.Join(os.Getenv("GOBIN"), "gog")

	pkg := unit.(*Package)

	cont, err := v.containerForRepo(dir, unit, c)
	if err != nil {
		return nil, err
	}

	// Install VCS tools in Docker container.
	cont.Dockerfile = append(cont.Dockerfile, []byte("RUN apt-get -yq install git mercurial bzr subversion\n")...)

	cont.AddFiles = append(cont.AddFiles, [2]string{gogBinPath, "/usr/local/bin/gog"})
	cont.Cmd = []string{"bash", "-c", fmt.Sprintf("go get -v -t %s; gog %s", pkg.ImportPath, pkg.ImportPath)}

	cmd := container.Command{
		Container: *cont,
		Transform: func(in []byte) ([]byte, error) {
			var o gog.Output
			err := json.Unmarshal(in, &o)
			if err != nil {
				return nil, err
			}

			o2 := grapher2.Output{
				Symbols: make([]*graph.Symbol, len(o.Symbols)),
				Refs:    make([]*graph.Ref, len(o.Refs)),
				Docs:    make([]*graph.Doc, len(o.Docs)),
			}

			for i, gs := range o.Symbols {
				o2.Symbols[i], err = v.convertGoSymbol(gs, c)
				if err != nil {
					return nil, err
				}
			}
			for i, gr := range o.Refs {
				o2.Refs[i], err = v.convertGoRef(gr, c)
				if err != nil {
					return nil, err
				}
			}
			for i, gd := range o.Docs {
				o2.Docs[i], err = v.convertGoDoc(gd, c)
				if err != nil {
					return nil, err
				}
			}

			return json.Marshal(o2)
		},
	}

	return &cmd, nil
}

// SymbolData is extra Go-specific data about a symbol.
type SymbolData struct {
	gog.SymbolInfo

	// PackageImportPath is the import path of the package containing this
	// symbol (if this symbol is not a package). If this symbol is a package,
	// PackageImportPath is its own import path.
	PackageImportPath string `json:",omitempty"`
}

func (v *goVersion) convertGoSymbol(gs *gog.Symbol, c *config.Repository) (*graph.Symbol, error) {
	resolvedTarget, err := v.resolveGoImportDep(gs.SymbolKey.PackageImportPath, c)
	if err != nil {
		return nil, err
	}

	var path graph.SymbolPath
	if len(gs.Path) > 0 {
		path = graph.SymbolPath(strings.Join(gs.Path, "/"))
	} else {
		path = "."
	}

	sym := &graph.Symbol{
		SymbolKey: graph.SymbolKey{
			Unit:     resolvedTarget.ToUnit,
			UnitType: resolvedTarget.ToUnitType,
			Path:     path,
		},

		Name: gs.Name,
		Kind: graph.SymbolKind(gog.GeneralKindMap[gs.Kind]),

		File:     gs.File,
		DefStart: gs.DeclSpan[0],
		DefEnd:   gs.DeclSpan[1],

		Exported: gs.SymbolInfo.Exported,
		Test:     strings.HasSuffix(gs.File, "_test.go"),
	}

	d := SymbolData{
		PackageImportPath: gs.SymbolKey.PackageImportPath,
		SymbolInfo:        gs.SymbolInfo,
	}
	sym.Data, err = json.Marshal(d)
	if err != nil {
		return nil, err
	}

	if sym.Kind == "func" {
		sym.Callable = true
	}

	return sym, nil
}

func (v *goVersion) convertGoRef(gr *gog.Ref, c *config.Repository) (*graph.Ref, error) {
	resolvedTarget, err := v.resolveGoImportDep(gr.Symbol.PackageImportPath, c)
	if err != nil {
		return nil, err
	}
	if resolvedTarget == nil {
		return nil, nil
	}
	return &graph.Ref{
		SymbolRepo:     uriOrEmpty(resolvedTarget.ToRepoCloneURL),
		SymbolPath:     graph.SymbolPath(strings.Join(gr.Symbol.Path, "/")),
		SymbolUnit:     resolvedTarget.ToUnit,
		SymbolUnitType: resolvedTarget.ToUnitType,
		Def:            gr.Def,
		File:           gr.File,
		Start:          gr.Span[0],
		End:            gr.Span[1],
	}, nil
}

func (v *goVersion) convertGoDoc(gd *gog.Doc, c *config.Repository) (*graph.Doc, error) {
	resolvedTarget, err := v.resolveGoImportDep(gd.PackageImportPath, c)
	if err != nil {
		return nil, err
	}
	return &graph.Doc{
		SymbolKey: graph.SymbolKey{
			Path:     graph.SymbolPath(strings.Join(gd.Path, "/")),
			Unit:     resolvedTarget.ToUnit,
			UnitType: resolvedTarget.ToUnitType,
		},
		Format: gd.Format,
		Data:   gd.Data,
		File:   gd.File,
		Start:  gd.Span[0],
		End:    gd.Span[1],
	}, nil
}

func uriOrEmpty(cloneURL string) repo.URI {
	if cloneURL == "" {
		return ""
	}
	return repo.MakeURI(cloneURL)
}
