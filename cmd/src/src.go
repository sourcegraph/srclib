package main

import (
	_ "sourcegraph.com/sourcegraph/srclib/dep"
	_ "sourcegraph.com/sourcegraph/srclib/scan"
	"sourcegraph.com/sourcegraph/srclib/src"
)

func main() {
	src.Main()
}
