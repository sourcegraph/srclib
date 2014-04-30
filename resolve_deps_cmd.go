package srcgraph

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"sourcegraph.com/sourcegraph/srcgraph/dep2"
	"sourcegraph.com/sourcegraph/srcgraph/task2"
)

func resolveDeps(args []string) {
	fs := flag.NewFlagSet("resolve-deps", flag.ExitOnError)
	jsonOutput := fs.Bool("json", false, "show JSON output")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: `+Name+` resolve-deps [options] [raw_dep_file.json...]

Resolves a repository's raw dependencies. If no files are specified, input is
read from stdin.

The options are:
`)
		fs.PrintDefaults()
		os.Exit(1)
	}
	fs.Parse(args)

	inputs := OpenInputFiles(fs.Args())
	defer CloseAll(inputs)

	context, err := NewJobContext(*Dir, task2.DefaultContext)
	if err != nil {
		log.Fatal(err)
	}

	var allRawDeps []*dep2.RawDependency
	for name, input := range inputs {
		if *Verbose {
			log.Printf("Reading raw deps from %q", name)
		}
		var rawDeps []*dep2.RawDependency
		err := json.NewDecoder(input).Decode(&rawDeps)
		if err != nil {
			log.Fatalf("%s: %s", name, err)
		}

		allRawDeps = append(allRawDeps, rawDeps...)
	}

	resolvedDeps, err := dep2.ResolveAll(allRawDeps, context.Repo, task2.DefaultContext)
	if err != nil {
		log.Fatal(err)
	}
	if resolvedDeps == nil {
		resolvedDeps = []*dep2.ResolvedDep{}
	}

	if *jsonOutput {
		PrintJSON(resolvedDeps, "")
	}
}
