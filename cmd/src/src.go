package main

import (
	_ "github.com/sourcegraph/srclib/dep2"
	_ "github.com/sourcegraph/srclib/scan"
	"github.com/sourcegraph/srclib/src"
)

func main() {
	src.Main()
}
