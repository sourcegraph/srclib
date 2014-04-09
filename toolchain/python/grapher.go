package python

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"sourcegraph.com/sourcegraph/graph"
	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/container"
	"sourcegraph.com/sourcegraph/srcgraph/grapher2"
	"sourcegraph.com/sourcegraph/srcgraph/task2"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
)

func init() {
	grapher2.Register(&fauxPackage{}, grapher2.DockerGrapher{defaultPythonEnv})
}

const srcRoot = "/src"

var grapherDockerfileTemplate = template.Must(template.New("").Parse(`FROM dockerfile/java
RUN apt-get update
RUN apt-get install -qy curl
RUN apt-get install -qy git
RUN apt-get install -qy {{.Python}}
RUN ln -s $(which {{.Python}}) /usr/bin/python
RUN curl https://raw.github.com/pypa/pip/master/contrib/get-pip.py > get-pip.py
RUN python get-pip.py
RUN pip install virtualenv

# Pysonar
RUN apt-get install -qy maven
RUN git clone --depth 1 --branch v0.0 https://github.com/sourcegraph/pysonar2.git /pysonar2
WORKDIR pysonar2
RUN mvn clean package
WORKDIR /

# Set up virtualenv (will contain dependencies)
RUN virtualenv /venv
`))

var grapherDockerCmdTemplate = template.Must(template.New("").Parse(`
/venv/bin/pip install -e {{.SrcDir}} 1>&2 || /venv/bin/pip install -r {{.SrcDir}}/requirements.txt 1>&2;
java {{.JavaOpts}} -classpath /pysonar2/target/pysonar-2.0-SNAPSHOT.jar org.yinwang.pysonar.JSONDump {{.SrcDir}} '{{.IncludePaths}}' '';
`))

func (p *pythonEnv) grapherDockerfile() []byte {
	var buf bytes.Buffer
	grapherDockerfileTemplate.Execute(&buf, struct {
		Python string
		SrcDir string
	}{
		Python: p.PythonVersion,
		SrcDir: srcRoot,
	})
	return buf.Bytes()
}

func (p *pythonEnv) stdLibDir() string {
	return fmt.Sprintf("/usr/lib/%s", p.PythonVersion)
}

func (p *pythonEnv) sitePackagesDir(virtualenvRoot string) string {
	return filepath.Join(virtualenvRoot, "lib", p.PythonVersion, "site-packages")
}

func (p *pythonEnv) grapherCmd() []string {
	javaOpts := os.Getenv("PYGRAPH_JAVA_OPTS")
	inclpaths := []string{srcRoot, p.stdLibDir(), p.sitePackagesDir("/venv")}

	var buf bytes.Buffer
	grapherDockerCmdTemplate.Execute(&buf, struct {
		JavaOpts     string
		SrcDir       string
		IncludePaths string
	}{
		JavaOpts:     javaOpts,
		SrcDir:       srcRoot,
		IncludePaths: strings.Join(inclpaths, ":"),
	})
	return []string{"/bin/bash", "-c", buf.String()}
}

func (p *pythonEnv) BuildGrapher(dir string, unit unit.SourceUnit, c *config.Repository, x *task2.Context) (*container.Command, error) {
	return &container.Command{
		Container: container.Container{
			RunOptions: []string{"-v", dir + ":" + srcRoot},
			Dockerfile: p.grapherDockerfile(),
			Cmd:        p.grapherCmd(),
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
				sym := convertSym(psym, srcRoot)
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
				ref := convertRef(pref, srcRoot, c)
				if _, exists := selfRefs[*ref]; !exists {
					o2.Refs = append(o2.Refs, ref)
				}
			}
			for _, pdoc := range o.Docs {
				o2.Docs = append(o2.Docs, convertDoc(pdoc, srcRoot))
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
