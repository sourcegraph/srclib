package python

import (
	"bytes"
	"text/template"

	"sourcegraph.com/sourcegraph/srcgraph/repo"
	"sourcegraph.com/sourcegraph/srcgraph/toolchain"
)

const (
	srcRoot        = "/src"
	stdLibRepo     = repo.URI("hg.python.org/cpython")
	pythonUnitType = "python"
)

type pythonEnv struct {
	PythonVersion  string
	Python3Version string
	PydepVersion   string
}

var defaultPythonEnv = &pythonEnv{
	PythonVersion:  "python2.7",
	Python3Version: "python3.3",
	PydepVersion:   "bd61d1a16f696b90828e198a610da7aae10b8ac2",
}

func init() {
	toolchain.Register("python", defaultPythonEnv)
}

type DistPackage struct {
	ProjectName string
	Files       []string
}

func (p *DistPackage) Name() string {
	return p.ProjectName
}

func (p *DistPackage) RootDir() string {
	return "."
}

func (p *DistPackage) Paths() []string {
	return p.Files
}

// pydep data structures

type pkgInfo struct {
	ProjectName string   `json:"project_name,omitempty"`
	Version     string   `json:"version,omitempty"`
	RepoURL     string   `json:"repo_url,omitempty"`
	Packages    []string `json:"packages,omitempty"`
	Modules     []string `json:"modules,omitempty"`
	Scripts     []string `json:"scripts,omitempty"`
	Author      string   `json:"author,omitempty"`
	Description string   `json:"description,omitempty"`
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
