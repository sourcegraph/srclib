package python

import (
	"sourcegraph.com/sourcegraph/srcgraph/repo"
	"sourcegraph.com/sourcegraph/srcgraph/toolchain"
)

const srcRoot = "/src"
const stdLibRepo = repo.URI("hg.python.org/cpython")

type pythonEnv struct {
	PythonVersion  string
	Python3Version string
	PydepVersion   string
}

var defaultPythonEnv = &pythonEnv{
	PythonVersion:  "python2.7",
	Python3Version: "python3.3",
	PydepVersion:   "65604616d5ea53e98475d89e6d9891f8f627edda",
}

func init() {
	toolchain.Register("python", defaultPythonEnv)
}

type FauxPackage struct {
	Files []string
}

func (p *FauxPackage) Name() string {
	return "python-faux-package"
}

func (p *FauxPackage) RootDir() string {
	return "."
}

func (p *FauxPackage) Paths() []string {
	return p.Files
}
