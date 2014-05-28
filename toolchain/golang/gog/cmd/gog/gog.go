package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/build"
	"log"
	"os"
	"path/filepath"
	"strings"

	"sourcegraph.com/sourcegraph/srcgraph/toolchain/golang/gog"
)

var buildTags = flag.String("tags", "", "a list of build tags to consider satisfied")
var srcImports = flag.Bool("src", false, "use source (not compiled binary pkgs) for analysis")

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: gog [options] [packages]\n\n")
		fmt.Fprintf(os.Stderr, "Graphs the named Go package.\n\n")
		fmt.Fprintf(os.Stderr, "The options are:\n")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr)
		fmt.Fprintf(os.Stderr, "For more about specifying packages, see 'go help packages'.\n")
		os.Exit(1)
	}
	flag.Parse()

	log.SetFlags(0)

	config := &gog.Default
	config.SourceImports = *srcImports

	if tags := strings.Split(*buildTags, ","); *buildTags != "" {
		build.Default.BuildTags = tags
		config.Build.BuildTags = tags
		log.Printf("Using build tags: %q", tags)
	}

	var importUnsafe bool
	for _, a := range flag.Args() {
		if a == "unsafe" {
			importUnsafe = true
			break
		}
	}

	extraArgs, err := config.FromArgs(flag.Args(), true)
	if err != nil {
		log.Fatal(err)
	}
	if len(extraArgs) > 0 {
		flag.Usage()
	}

	if importUnsafe {
		// Special-case "unsafe" because go/loader does not let you load it
		// directly.
		if config.ImportPkgs == nil {
			config.ImportPkgs = make(map[string]bool)
		}
		config.ImportPkgs["unsafe"] = true
	}

	prog, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	g := gog.New(prog)

	err = g.GraphImported()
	if err != nil {
		log.Fatal(err)
	}

	for _, gs := range g.Output.Symbols {
		if gs.File == "" {
			log.Printf("no file %+v", gs)
		}
		gs.File = relPath(gs.File)
	}
	for _, gr := range g.Output.Refs {
		gr.File = relPath(gr.File)
	}
	for _, gd := range g.Output.Docs {
		if gd.File != "" {
			gd.File = relPath(gd.File)
		}
	}

	err = json.NewEncoder(os.Stdout).Encode(g.Output)
	if err != nil {
		log.Fatal(err)
	}
}

var cwd string

func init() {
	var err error
	cwd, err = os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
}

func relPath(path string) string {
	rp, err := filepath.Rel(cwd, path)
	if err != nil {
		log.Fatalf("Failed to make path %q relative to %q: %s", path, cwd, err)
	}
	return rp
}
