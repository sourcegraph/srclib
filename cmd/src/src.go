package main

import (
	"os"

	_ "sourcegraph.com/sourcegraph/srclib/dep"
	_ "sourcegraph.com/sourcegraph/srclib/scan"
	"sourcegraph.com/sourcegraph/srclib/src"
)

func main() {
	if err := src.Main(); err != nil {
		os.Exit(1)
	}
}
