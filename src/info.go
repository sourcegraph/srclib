package src

import (
	"flag"
	"fmt"
	"log"
	"os"
	"reflect"

	"github.com/kr/text"
	"github.com/sourcegraph/srclib/build"
	"github.com/sourcegraph/srclib/buildstore"
	"github.com/sourcegraph/srclib/config"
	"github.com/sourcegraph/srclib/dep2"
	"github.com/sourcegraph/srclib/graph"
	"github.com/sourcegraph/srclib/grapher2"
	"github.com/sourcegraph/srclib/scan"
	"github.com/sourcegraph/srclib/toolchain"
	"github.com/sourcegraph/srclib/unit"
)

func info(args []string) {
	fs := flag.NewFlagSet("help", flag.ExitOnError)
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

	log.Printf("Toolchains (%d)", len(toolchain.Toolchains))
	for tcName, _ := range toolchain.Toolchains {
		log.Printf(" - %s", tcName)
	}
	log.Println()

	log.Printf("Config global sections (%d)", len(config.Globals))
	for name, typ := range config.Globals {
		log.Printf(" - %s (type %T)", name, typ)
	}
	log.Println()

	log.Printf("Source units (%d)", len(unit.Types))
	for name, typ := range unit.Types {
		log.Printf(" - %s (type %T)", name, typ)
	}
	log.Println()

	log.Printf("Symbol formatters (%d)", len(graph.MakeSymbolFormatters))
	for unitType, f := range graph.MakeSymbolFormatters {
		log.Printf(" - %s (type %s)", unitType, reflect.TypeOf(f).Out(0))
	}
	log.Println()

	log.Printf("Scanners (%d)", len(scan.Scanners))
	for name, _ := range scan.Scanners {
		log.Printf(" - %s", name)
	}
	log.Println()

	log.Printf("Graphers (%d)", len(grapher2.Graphers))
	for typ, _ := range grapher2.Graphers {
		log.Printf(" - %s source units", unit.TypeNames[typ])
	}
	log.Println()

	log.Printf("Dependency raw listers (%d)", len(dep2.Listers))
	for typ, _ := range dep2.Listers {
		log.Printf(" - %s source units", unit.TypeNames[typ])
	}
	log.Println()

	log.Printf("Dependency resolvers (%d)", len(dep2.Resolvers))
	for typ, _ := range dep2.Resolvers {
		log.Printf(" - %q raw dependencies", typ)
	}
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
	log.Println()

	log.Printf("------------------")
	log.Println()
	log.Printf("System information:")
	log.Printf(" - docker version:\n%s", text.Indent(cmdOutput("docker", "version"), "         "))
}
