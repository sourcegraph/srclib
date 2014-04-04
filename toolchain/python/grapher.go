package python

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"sourcegraph.com/sourcegraph/graph"
	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/container"
	"sourcegraph.com/sourcegraph/srcgraph/grapher2"
	"sourcegraph.com/sourcegraph/srcgraph/task2"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
)

func init() {
	grapher2.Register(&fauxPackage{}, grapher2.DockerGrapher{&pythonGrapherBuilder{}})
}

type pythonGrapherBuilder struct{}

const java = "java"

func (p *pythonGrapherBuilder) BuildGrapher(dir string, unit unit.SourceUnit, c *config.Repository, x *task2.Context) (*container.Command, error) {
	containerSrcDir := "/src"

	inclpaths := []string{} // TODO
	srcpath := containerSrcDir

	cmd := []string{java}
	if launcherOpts := os.Getenv("PYGRAPH_JAVA_OPTS"); launcherOpts != "" {
		cmd = append(cmd, strings.Split(launcherOpts, " ")...)
	}
	cmd = append(cmd, "-classpath", "target/pysonar-2.0-SNAPSHOT.jar", "org.yinwang.pysonar.JSONDump", srcpath, strings.Join(inclpaths, ":"), "")

	return &container.Command{
		Container: container.Container{
			Dockerfile: []byte(pysonarDockerfile),
			RunOptions: []string{"-v", dir + ":" + containerSrcDir},
			Cmd:        cmd,
		},
		Transform: func(orig []byte) ([]byte, error) {
			var o pysonarData
			err := json.Unmarshal(orig, &o)
			if err != nil {
				return nil, err
			}
			o2 := grapher2.Output{
				Symbols: make([]*graph.Symbol, 0),
				Refs:    make([]*graph.Ref, 0),
				Docs:    make([]*graph.Doc, 0),
			}

			selfRefs := make(map[graph.Ref]struct{})
			for _, psym := range o.Syms {
				sym := convertSym(psym, containerSrcDir)
				if sym.Path != "" {
					o2.Symbols = append(o2.Symbols, sym)
				}

				if sym.File != "" && sym.IdentStart != sym.IdentEnd {
					selfRef := symToSelfRef(sym)
					selfRefs[*selfRef] = struct{}{}
					o2.Refs = append(o2.Refs, selfRef)
				}
			}
			for _, pref := range o.Refs {
				ref := convertRef(pref, containerSrcDir, c)
				if _, exists := selfRefs[*ref]; !exists {
					o2.Refs = append(o2.Refs, ref)
				}
			}
			for _, pdoc := range o.Docs {
				o2.Docs = append(o2.Docs, convertDoc(pdoc, containerSrcDir))
			}

			return json.Marshal(o2)
		},
	}, nil
}

func symToSelfRef(sym *graph.Symbol) *graph.Ref {
	return &graph.Ref{
		SymbolRepo: "",
		SymbolPath: sym.Path,

		Repo:  "", // purposefully omitted
		File:  sym.File,
		Start: sym.IdentStart,
		End:   sym.IdentEnd,
	}
}

func convertSym(pySym *pySym, containerSrcDir string) *graph.Symbol {
	relpath, _ := filepath.Rel(containerSrcDir, pySym.Path)

	sym := &graph.Symbol{
		SymbolKey: graph.SymbolKey{
			Path: graph.SymbolPath(relpath),
		},
		Name:       pySym.Name,
		File:       pySym.File,
		IdentStart: pySym.IdentStart,
		IdentEnd:   pySym.IdentEnd,
		DefStart:   pySym.DefStart,
		DefEnd:     pySym.DefEnd,
		Exported:   pySym.Exported,
		Callable:   callablePythonSymbolKinds[pySym.Kind],
	}

	if pySym.Exported {
		components := strings.Split(string(sym.Path), "/")
		if len(components) == 1 {
			sym.SpecificPath = components[0]
		} else {
			// take the last 2 path components
			sym.SpecificPath = components[len(components)-2] + "." + components[len(components)-1]
		}
	} else {
		sym.SpecificPath = pySym.Name
	}

	if pySym.FuncData != nil {
		sym.TypeExpr = pySym.FuncData.Signature
	}

	if pySym.Kind != "MODULE" {
		sym.SpecificKind = strings.ToLower(pySym.Kind)
	} else {
		if strings.HasSuffix(pySym.File, "__init__.py") {
			sym.SpecificKind = Package
		} else {
			sym.SpecificKind = Module
		}
	}

	sym.Kind = py2sgSymKindMap[sym.SpecificKind]
	sym.SpecificKind = py2SpecificSymKindMap[sym.SpecificKind]

	return sym
}

func convertRef(pyRef *pyRef, containerSrcDir string, c *config.Repository) *graph.Ref {
	// TODO: handle dependencies
	return &graph.Ref{
		SymbolRepo: c.URI,                       // might be incorrect
		SymbolPath: graph.SymbolPath(pyRef.Sym), // might be incorrect

		File:  pyRef.File,
		Start: pyRef.Start,
		End:   pyRef.End,
	}
}

func convertDoc(pyDoc *pyDoc, containerSrcDir string) *graph.Doc {
	// TODO: handle null byte (\x00) in doc body?
	relpath, _ := filepath.Rel(containerSrcDir, pyDoc.Sym)
	return &graph.Doc{
		SymbolKey: graph.SymbolKey{
			// TODO: more fields
			Path: graph.SymbolPath(relpath),
		},
		Data:  formatDocs(pyDoc.Body),
		File:  pyDoc.File,
		Start: pyDoc.Start,
		End:   pyDoc.End,
	}
}

type pysonarData struct {
	Syms []*pySym `json:"syms"`
	Refs []*pyRef `json:"refs"`
	Docs []*pyDoc `json:"docs"`
}

type pySym struct {
	Path       string `json:"path"`
	Name       string `json:"name"`
	File       string `json:"file"`
	IdentStart int    `json:"identStart"`
	IdentEnd   int    `json:"identEnd"`
	DefStart   int    `json:"defStart"`
	DefEnd     int    `json:"defEnd"`
	Exported   bool   `json:"exported"`
	Kind       string `json:"kind"`
	FuncData   *struct {
		Signature string `json:"signature"`
	} `json:"funcData,omitempty"`
}

type pyRef struct {
	Sym     string `json:"sym"`
	File    string `json:"file"`
	Start   int    `json:"start"`
	End     int    `json:"end"`
	Builtin bool   `json:"builtin"`
}

type pyDoc struct {
	Sym   string `json:"sym"`
	File  string `json:"file"`
	Body  string `json:"body"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}

const pysonarDockerfile = `FROM dockerfile/java
RUN apt-get update
RUN apt-get install -qy maven

RUN git clone --depth 1 --branch v0.0 https://github.com/sourcegraph/pysonar2.git /pysonar2
WORKDIR pysonar2
RUN mvn clean package
`
