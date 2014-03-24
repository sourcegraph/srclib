package golang

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"strings"

	"sourcegraph.com/sourcegraph/graph"
	"sourcegraph.com/sourcegraph/repo"
	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/container"
	"sourcegraph.com/sourcegraph/srcgraph/grapher2"
	"sourcegraph.com/sourcegraph/srcgraph/task2"
	"sourcegraph.com/sourcegraph/srcgraph/toolchain/golang/gog"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
)

func init() {
	grapher2.Register(Package{}, grapher2.DockerGrapher{defaultGoVersion})
}

func (v *goVersion) BuildGrapher(dir string, unit unit.SourceUnit, c *config.Repository, x *task2.Context) (*container.Command, error) {
	gogBinPath := filepath.Join(os.Getenv("GOBIN"), "gog")

	dockerfile, err := v.baseDockerfile()
	if err != nil {
		return nil, err
	}

	// Install VCS tools in Docker container.
	dockerfile = append(dockerfile, []byte("RUN apt-get -yq install git mercurial bzr subversion\n")...)

	pkg := unit.(*Package)
	var preCmd []byte
	preCmd = append(preCmd, []byte(fmt.Sprintf("\nRUN go get -v -t %s\nRUN go get -d -v -t %s\n", pkg.ImportPath, pkg.ImportPath))...)

	goConfig := v.goConfig(c)
	containerDir := filepath.Join(containerGOPATH, "src", goConfig.BaseImportPath)
	cmd := container.Command{
		Container: container.Container{
			Dockerfile:       dockerfile,
			AddDirs:          [][2]string{{dir, containerDir}},
			AddFiles:         [][2]string{{gogBinPath, "/usr/local/bin/gog"}},
			PreCmdDockerfile: preCmd,
			Cmd:              []string{"gog", pkg.ImportPath},
			Dir:              containerDir,
			Stderr:           x.Stderr,
			Stdout:           x.Stdout,
		},
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

func (v *goVersion) convertGoSymbol(gs *gog.Symbol, c *config.Repository) (*graph.Symbol, error) {
	resolvedTarget, err := v.resolveGoImportDep(gs.SymbolKey.PackageImportPath, c)
	if err != nil {
		return nil, err
	}

	sym := &graph.Symbol{
		SymbolKey: graph.SymbolKey{
			Unit:     gs.PackageImportPath,
			Repo:     repo.MakeURI(resolvedTarget.ToRepoCloneURL),
			UnitType: "go",
			Path:     graph.SymbolPath(strings.Join(gs.Path, "/")),
		},

		Name:         gs.Name,
		SpecificPath: gs.Name, // TODO!(sqs)
		TypeExpr:     gs.Description,
		Kind:         graph.SymbolKind(gog.GeneralKindMap[gs.Kind]),
		SpecificKind: gs.Kind,

		File:     gs.File,
		DefStart: gs.DeclSpan[0],
		DefEnd:   gs.DeclSpan[1],

		Exported: gs.Exported,
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
	return &graph.Ref{
		SymbolRepo:     repo.MakeURI(resolvedTarget.ToRepoCloneURL),
		SymbolPath:     graph.SymbolPath(strings.Join(gr.Symbol.Path, "/")),
		SymbolUnit:     gr.Symbol.PackageImportPath,
		SymbolUnitType: "go",
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
			Repo:     repo.MakeURI(resolvedTarget.ToRepoCloneURL),
			Path:     graph.SymbolPath(strings.Join(gd.Path, "/")),
			Unit:     gd.PackageImportPath,
			UnitType: "go",
		},
		Format: gd.Format,
		Data:   gd.Data,
		File:   gd.File,
		Start:  gd.Span[0],
		End:    gd.Span[1],
	}, nil
}
