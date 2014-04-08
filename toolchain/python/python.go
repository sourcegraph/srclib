package python

import "sourcegraph.com/sourcegraph/srcgraph/toolchain"

func init() {
	toolchain.Register("python", defaultPythonEnv)
}

type pythonEnv struct {
	PythonVersion string
}

var defaultPythonEnv = &pythonEnv{
	PythonVersion: "python2.7",
}
