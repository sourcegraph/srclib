// Package all_toolchains imports and registers all known toolchains as a side
// effect of being imported. It should be imported
// for side effects only (`import _
// "sourcegraph.com/sourcegraph/srcgraph/toolchain/all_toolchains"`).
package all_toolchains

import (
	_ "sourcegraph.com/sourcegraph/srcgraph/toolchain/golang"
	_ "sourcegraph.com/sourcegraph/srcgraph/toolchain/python"
	// _ "sourcegraph.com/sourcegraph/srcgraph/toolchain/ruby"
	_ "sourcegraph.com/sourcegraph/srcgraph/config/overrides"
	_ "sourcegraph.com/sourcegraph/srcgraph/toolchain/javascript"
)
