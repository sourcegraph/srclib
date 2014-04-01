package python

import "sourcegraph.com/sourcegraph/srcgraph/toolchain"

func init() {
	toolchain.Register("python", struct{}{})
}
