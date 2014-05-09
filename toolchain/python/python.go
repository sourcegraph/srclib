package python

import "sourcegraph.com/sourcegraph/srcgraph/toolchain"

type pythonEnv struct {
	PythonVersion  string
	Python3Version string
}

var defaultPythonEnv = &pythonEnv{
	PythonVersion:  "python2.7",
	Python3Version: "python3.3",
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
