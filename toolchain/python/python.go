package python

import (
	"bytes"
	"path/filepath"
	"text/template"

	"sourcegraph.com/sourcegraph/srcgraph/repo"
	"sourcegraph.com/sourcegraph/srcgraph/toolchain"
)

const (
	srcRoot    = "/src"
	stdLibRepo = repo.URI("hg.python.org/cpython")
)

// Taken from hg.python.org/cpython's setup.py
var stdLibUnit = &DistPackage{
	ProjectName: "Python",
	ProjectDescription: `A high-level object-oriented programming language

Python is an interpreted, interactive, object-oriented programming
language. It is often compared to Tcl, Perl, Scheme or Java.

Python combines remarkable power with very clear syntax. It has
modules, classes, exceptions, very high level dynamic data types, and
dynamic typing. There are interfaces to many system calls and
libraries, as well as to various windowing systems (X11, Motif, Tk,
Mac, MFC). New built-in modules are easily written in C or C++. Python
is also usable as an extension language for applications that need a
programmable interface.

The Python implementation is portable: it runs on many brands of UNIX,
on Windows, DOS, Mac, Amiga... If your favorite system isn't
listed here, it may still be supported, if there's a C compiler for
it. Ask around on comp.lang.python -- or just try compiling Python
yourself.`,
	RootDirectory: "Lib",
	Files:         nil, // should be filled in when needed
}

type pythonEnv struct {
	PythonVersion   string
	Python3Version  string
	PydepVersion    string
	PySonar2Version string
}

var defaultPythonEnv = &pythonEnv{
	PythonVersion:   "python2.7",
	Python3Version:  "python3.3",
	PydepVersion:    "debfd0e681c3b60e33eec237a4473aed1f767004",
	PySonar2Version: "1b152a16d1292b66280e60047a8dbdbfc86a103b",
}

func init() {
	toolchain.Register("python", defaultPythonEnv)
}

const DistPackageDisplayName = "PipPackage"

type DistPackage struct {
	// Name of the DistPackage as defined in setup.py. E.g., Django, Flask, etc.
	ProjectName string

	// Description of the DistPackage (extracted from its setup.py). This may be empty if derived from a requirement.
	ProjectDescription string

	// The root directory relative to the repository root that contains the setup.py. This may be empty if this
	// DistPackage is derived from a requirement (there is no way to recover a Python distUtils package's location in
	// its source repository without accessing the source repository itself).
	RootDirectory string

	// The files in the package. This may be empty (it is only necessary for computing blame).
	Files []string
}

func (p *DistPackage) Name() string {
	return p.ProjectName
}

func (p *DistPackage) RootDir() string {
	return p.RootDirectory
}

func (p *DistPackage) Paths() []string {
	paths := make([]string, len(p.Files))
	for i, f := range p.Files {
		paths[i] = filepath.Join(p.RootDirectory, f)
	}
	return paths
}

// NameInRepository implements unit.Info.
func (p *DistPackage) NameInRepository(defining repo.URI) string { return p.Name() }

// GlobalName implements unit.Info.
func (p *DistPackage) GlobalName() string { return p.Name() }

// Description implements unit.Info.
func (p *DistPackage) Description() string { return p.ProjectDescription }

// Type implements unit.Info.
func (p *DistPackage) Type() string { return "Python package" }

// pydep data structures

type pkgInfo struct {
	RootDir     string   `json:"rootdir,omitempty"`
	ProjectName string   `json:"project_name,omitempty"`
	Version     string   `json:"version,omitempty"`
	RepoURL     string   `json:"repo_url,omitempty"`
	Packages    []string `json:"packages,omitempty"`
	Modules     []string `json:"modules,omitempty"`
	Scripts     []string `json:"scripts,omitempty"`
	Author      string   `json:"author,omitempty"`
	Description string   `json:"description,omitempty"`
}

func (p pkgInfo) DistPackage() *DistPackage {
	return &DistPackage{
		ProjectName:        p.ProjectName,
		ProjectDescription: p.Description,
		RootDirectory:      p.RootDir,
	}
}

func (p pkgInfo) DistPackageWithFiles(files []string) *DistPackage {
	return &DistPackage{
		ProjectName:        p.ProjectName,
		ProjectDescription: p.Description,
		RootDirectory:      p.RootDir,
		Files:              files,
	}
}

type requirement struct {
	ProjectName string      `json:"project_name"`
	UnsafeName  string      `json:"unsafe_name"`
	Key         string      `json:"key"`
	Specs       [][2]string `json:"specs"`
	Extras      []string    `json:"extras"`
	RepoURL     string      `json:"repo_url"`
	Packages    []string    `json:"packages"`
	Modules     []string    `json:"modules"`
	Resolved    bool        `json:"resolved"`
	Type        string      `json:"type"`
}

func (r requirement) DistPackage() *DistPackage {
	return &DistPackage{
		ProjectName: r.ProjectName,
	}
}

func (l *pythonEnv) pydepDockerfile() ([]byte, error) {
	var buf bytes.Buffer
	template.Must(template.New("").Parse(pydepDockerfileTemplate)).Execute(&buf, l)
	return buf.Bytes(), nil
}

const pydepDockerfileTemplate = `FROM ubuntu:14.04
RUN apt-get update -qq
RUN apt-get install -qqy curl
RUN apt-get install -qqy git
RUN apt-get install -qqy {{.PythonVersion}}
RUN ln -s $(which {{.PythonVersion}}) /usr/bin/python
RUN curl https://raw.githubusercontent.com/pypa/pip/1.5.5/contrib/get-pip.py | python

RUN pip install git+git://github.com/sourcegraph/pydep.git@{{.PydepVersion}}
`
