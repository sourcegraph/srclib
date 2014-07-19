// Package all_toolchains imports and registers all known toolchains as a side
// effect of being imported. It should be imported
// for side effects only (`import _
// "github.com/sourcegraph/srclib/toolchain/all_toolchains"`).
package all_toolchains

import (
	_ "github.com/sourcegraph/srclib/config/overrides"
	_ "github.com/sourcegraph/srclib/toolchain/golang"
	_ "github.com/sourcegraph/srclib/toolchain/javascript"
	_ "github.com/sourcegraph/srclib/toolchain/python"
	_ "github.com/sourcegraph/srclib/toolchain/ruby"
)
