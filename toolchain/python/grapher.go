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
	"sourcegraph.com/sourcegraph/repo"
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
const stdLibRepo = repo.URI("hg.python.org/cpython")

var grapherDockerfileTemplate = template.Must(template.New("").Parse(`FROM dockerfile/java
RUN apt-get update
RUN apt-get install -qy curl
RUN apt-get install -qy git
RUN apt-get install -qy {{.Python}}
RUN ln -s $(which {{.Python}}) /usr/bin/python
RUN curl https://raw.github.com/pypa/pip/master/contrib/get-pip.py > get-pip.py
RUN python get-pip.py
RUN pip install virtualenv

# PyDep
RUN pip install git+git://github.com/sourcegraph/pydep@0.0

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
/venv/bin/pip install {{.SrcDir}} 1>&2 || /venv/bin/pip install -r {{.SrcDir}}/requirements.txt 1>&2;

# Compute requirements
REQDATA=$(pydep-run.py {{.SrcDir}});

# Compute graph
mkfifo /tmp/pysonar.err;
cat -v /tmp/pysonar.err &> /dev/null &  # bug: container hangs if we print this output
GRAPHDATA=$(java {{.JavaOpts}} -classpath /pysonar2/target/pysonar-2.0-SNAPSHOT.jar org.yinwang.pysonar.JSONDump {{.SrcDir}} '{{.IncludePaths}}' '' 2>/tmp/pysonar.err);

echo "{ \"graph\": $GRAPHDATA, \"reqs\": $REQDATA }";
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

func (p *pythonEnv) sitePackagesDir() string {
	return filepath.Join("/venv", "lib", p.PythonVersion, "site-packages")
}

func (p *pythonEnv) grapherCmd() []string {
	javaOpts := os.Getenv("PYGRAPH_JAVA_OPTS")
	inclpaths := []string{srcRoot, p.stdLibDir(), p.sitePackagesDir()}

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
			Stderr:     x.Stderr,
			Stdout:     x.Stdout,
		},
		Transform: func(orig []byte) ([]byte, error) {
			var o rawGraphData
			err := json.Unmarshal(orig, &o)
			if err != nil {
				return nil, err
			}

			o2 := grapher2.Output{
				Symbols: make([]*graph.Symbol, 0),
				Refs:    make([]*graph.Ref, 0),
				Docs:    make([]*graph.Doc, 0),
			}

			selfrefs := make(map[graph.Ref]struct{})
			for _, psym := range o.Graph.Syms {
				sym, selfref, err := p.convertSym(psym, c, o.Reqs)
				if err != nil {
					return nil, err
				}

				if sym != nil {
					o2.Symbols = append(o2.Symbols, sym)
				}
				if selfref != nil {
					selfrefs[*selfref] = struct{}{}
					o2.Refs = append(o2.Refs, selfref)
				}
			}
			for _, pref := range o.Graph.Refs {
				ref, err := p.convertRef(pref, c, o.Reqs)
				if err != nil {
					return nil, err
				}
				if _, exists := selfrefs[*ref]; !exists {
					o2.Refs = append(o2.Refs, ref)
				}
			}
			for _, pdoc := range o.Graph.Docs {
				doc, err := p.convertDoc(pdoc, c, o.Reqs)
				if err != nil {
					return nil, err
				}
				o2.Docs = append(o2.Docs, doc)
			}

			return json.Marshal(o2)
		},
	}, nil
}

func (p *pythonEnv) convertSym(pySym *pySym, c *config.Repository, reqs []requirement) (sym *graph.Symbol, selfref *graph.Ref, err error) {
	symKey, err := p.pysonarSymPathToSymKey(pySym.Path, c, reqs)
	if err != nil {
		return
	}

	sym = &graph.Symbol{
		SymbolKey:    *symKey,
		Name:         pySym.Name,
		File:         pySym.File,
		IdentStart:   pySym.IdentStart,
		IdentEnd:     pySym.IdentEnd,
		DefStart:     pySym.DefStart,
		DefEnd:       pySym.DefEnd,
		Exported:     pySym.Exported,
		Callable:     callableSymbolKinds[pySym.Kind],
		Kind:         symbolKinds[pySym.Kind],
		SpecificKind: symbolSpecificKinds[pySym.Kind],
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
	if pySym.Kind == "MODULE" && strings.HasSuffix(pySym.File, "__init__.py") {
		sym.SpecificKind = Package
		sym.Kind = graph.Package
	}

	if sym.File != "" && sym.IdentStart != sym.IdentEnd {
		var symFile string
		symFile, err = p.pysonarFilePathToFile(pySym.File)
		if err != nil {
			return
		}
		selfref = &graph.Ref{
			SymbolRepo:     symKey.Repo,
			SymbolUnitType: symKey.UnitType,
			SymbolUnit:     symKey.Unit,
			SymbolPath:     symKey.Path,
			Def:            true,

			Repo:     symKey.Repo,
			UnitType: symKey.UnitType,
			Unit:     symKey.Unit,

			File:  symFile,
			Start: sym.IdentStart,
			End:   sym.IdentEnd,
		}
	}

	return
}

func (p *pythonEnv) convertRef(pyRef *pyRef, c *config.Repository, reqs []requirement) (*graph.Ref, error) {
	symKey, err := p.pysonarSymPathToSymKey(pyRef.Sym, c, reqs)
	if err != nil {
		return nil, err
	}
	refRepo, refFile, err := p.pysonarFilePathToRepoAndFile(pyRef.File, c, reqs)
	if err != nil {
		return nil, err
	}

	return &graph.Ref{
		SymbolRepo:     symKey.Repo,
		SymbolUnitType: symKey.UnitType,
		SymbolUnit:     symKey.Unit,
		SymbolPath:     symKey.Path,
		Def:            false,

		Repo:     refRepo,
		UnitType: unit.Type(&fauxPackage{}),
		Unit:     (&fauxPackage{}).Name(),

		File:  refFile,
		Start: pyRef.Start,
		End:   pyRef.End,
	}, nil
}

func (p *pythonEnv) convertDoc(pyDoc *pyDoc, c *config.Repository, reqs []requirement) (*graph.Doc, error) {
	// TODO: handle null byte (\x00) in doc body?
	symKey, err := p.pysonarSymPathToSymKey(pyDoc.Sym, c, reqs)
	if err != nil {
		return nil, err
	}
	docFile, err := p.pysonarFilePathToFile(pyDoc.File)
	if err != nil {
		return nil, err
	}
	return &graph.Doc{
		SymbolKey: *symKey,
		Format:    "", // TODO
		Data:      formatDocs(pyDoc.Body),
		File:      docFile,
		Start:     pyDoc.Start,
		End:       pyDoc.End,
	}, nil
}

func (p *pythonEnv) pysonarFilePathToFile(pth string) (string, error) {
	if newpath, err := filepath.Rel(srcRoot, pth); err == nil {
		return newpath, nil
	} else if newpath, err := filepath.Rel(p.sitePackagesDir(), pth); err == nil {
		return newpath, nil
	} else if newpath, err := filepath.Rel(p.stdLibDir(), pth); err == nil {
		return newpath, nil
	} else {
		return "", fmt.Errorf("Could not relativize file path %s", pth)
	}
}

func (p *pythonEnv) pysonarFilePathToRepoAndFile(pth string, c *config.Repository, reqs []requirement) (repo.URI, string, error) {
	if relpath, err := filepath.Rel(srcRoot, pth); err == nil {
		return c.URI, relpath, nil
	} else if relpath, err := filepath.Rel(p.sitePackagesDir(), pth); err == nil {
		var foundReq *requirement
	FindReq:
		for _, req := range reqs {
			for _, pkg := range req.Packages {
				pkgpath := strings.Replace(pkg, ".", "/", -1)
				if _, err := filepath.Rel(pkgpath, relpath); err == nil {
					foundReq = &req
					break FindReq
				}
			}
			for _, mod := range req.Modules {
				modpath := mod + ".py"
				if _, err := filepath.Rel(modpath, relpath); err == nil {
					foundReq = &req
					break FindReq
				}
			}
		}
		if foundReq == nil {
			return "", "", fmt.Errorf("Could not resolve repo URL for file path %s", pth)
		}
		return repo.MakeURI(foundReq.RepoURL), relpath, nil
	} else if relpath, err := filepath.Rel(p.stdLibDir(), pth); err == nil {
		return stdLibRepo, relpath, nil
	} else {
		return "", "", fmt.Errorf("Could not resolve repo URL for file path %s", pth)
	}
}

func (p *pythonEnv) pysonarSymPathToSymKey(pth string, c *config.Repository, reqs []requirement) (*graph.SymbolKey, error) {
	fauxUnit := &fauxPackage{}
	if relpath, err := filepath.Rel(srcRoot, pth); err == nil {
		return &graph.SymbolKey{
			Repo:     c.URI,
			UnitType: unit.Type(fauxUnit),
			Unit:     fauxUnit.Name(),
			Path:     graph.SymbolPath(relpath),
		}, nil
	} else if relpath, err := filepath.Rel(p.sitePackagesDir(), pth); err == nil {
		var foundReq *requirement
	FindReq:
		for _, req := range reqs {
			for _, pkg := range req.Packages {
				pkgpath := strings.Replace(pkg, ".", "/", -1)
				if _, err := filepath.Rel(pkgpath, relpath); err == nil {
					foundReq = &req
					break FindReq
				}
			}
			for _, mod := range req.Modules {
				modpath := mod
				if _, err := filepath.Rel(modpath, relpath); err == nil {
					foundReq = &req
					break FindReq
				}
			}
		}
		if foundReq == nil {
			return nil, fmt.Errorf("Could not find requirement matching path %s", pth)
		}

		return &graph.SymbolKey{
			Repo:     repo.MakeURI(foundReq.RepoURL),
			UnitType: unit.Type(fauxUnit),
			Unit:     fauxUnit.Name(),
			Path:     graph.SymbolPath(relpath),
		}, nil
	} else if relpath, err := filepath.Rel(p.stdLibDir(), pth); err == nil {
		return &graph.SymbolKey{
			Repo:     stdLibRepo,
			UnitType: unit.Type(fauxUnit),
			Unit:     fauxUnit.Name(),
			Path:     graph.SymbolPath(relpath),
		}, nil
	} else {
		return nil, fmt.Errorf("Could not find requirement matching path %s", pth)
	}
}

type rawGraphData struct {
	Graph struct {
		Syms []*pySym `json:"syms"`
		Refs []*pyRef `json:"refs"`
		Docs []*pyDoc `json:"docs"`
	} `json:"graph"`
	Reqs []requirement `json:"reqs"`
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
