package python

import "sourcegraph.com/sourcegraph/srcgraph/toolchain"

type pythonEnv struct {
	PythonVersion string
}

var defaultPythonEnv = &pythonEnv{
	PythonVersion: "python2.7",
}

func init() {
	toolchain.Register("python", defaultPythonEnv)
}

type fauxPackage struct {
	Files []string
}

func (p *fauxPackage) Name() string {
	return "python-faux-package"
}

func (p *fauxPackage) RootDir() string {
	return "."
}

func (p *fauxPackage) Paths() []string {
	return p.Files
}
