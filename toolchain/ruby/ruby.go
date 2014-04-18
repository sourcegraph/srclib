package ruby

import "sourcegraph.com/sourcegraph/srcgraph/toolchain"

const srcRoot = "/src" // path to source in container

func init() {
	toolchain.Register("ruby", defaultRubyEnv)
}

type rubyEnv struct {
	Ruby        string
	RDepVersion string
}

var defaultRubyEnv = &rubyEnv{
	Ruby:        "ruby2.0",
	RDepVersion: "0.0.4c",
}
