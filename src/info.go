package src

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/sourcegraph/srclib/build"
	"github.com/sourcegraph/srclib/buildstore"
	"github.com/sourcegraph/srclib/toolchain"
)

func info(args []string) {
	fs := flag.NewFlagSet("info", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: `+Name+` info

Shows information about enabled capabilities in this tool as well as system
information.

The options are:
`)
		fs.PrintDefaults()
		os.Exit(1)
	}
	fs.Parse(args)

	if fs.NArg() != 0 {
		fs.Usage()
	}

	log.Printf("srclib version %s\n", Version)
	log.Println("https://sourcegraph.com/sourcegraph/srclib")
	log.Println()

	log.Printf("SRCLIBPATH=%q", toolchain.SrclibPath)
	log.Println()

	log.Printf("Build data types (%d)", len(buildstore.DataTypes))
	for name, _ := range buildstore.DataTypes {
		log.Printf(" - %s", name)
	}
	log.Println()

	log.Printf("Build rule makers (%d)", len(build.RuleMakers))
	for name, _ := range build.RuleMakers {
		log.Printf(" - %s", name)
	}
}
