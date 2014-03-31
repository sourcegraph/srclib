package srcgraph

import (
	"flag"
	"fmt"
	"log"
	"os"

	"sourcegraph.com/sourcegraph/srcgraph/dep2"
	"sourcegraph.com/sourcegraph/srcgraph/task2"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
)

func listDeps(args []string) {
	fs := flag.NewFlagSet("list-deps", flag.ExitOnError)
	resolve := fs.Bool("resolve", false, "resolve deps and print resolutions")
	jsonOutput := fs.Bool("json", false, "show JSON output")
	r := AddRepositoryFlags(fs)
	rc := AddRepositoryConfigFlags(fs, r)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: `+Name+` list-deps [options] [unit...]

Lists a repository's raw (unresolved) dependencies. If unit(s) are specified,
only source units with matching IDs will have their dependencies listed.

The options are:
`)
		fs.PrintDefaults()
		os.Exit(1)
	}
	fs.Parse(args)
	sourceUnitSpecs := fs.Args()

	c := rc.GetRepositoryConfig(task2.DefaultContext)

	allRawDeps := []*dep2.RawDependency{}
	for _, u := range c.SourceUnits {
		if !SourceUnitMatchesArgs(sourceUnitSpecs, u) {
			continue
		}

		rawDeps, err := dep2.List(r.RootDir, u, c, task2.DefaultContext)
		if err != nil {
			log.Fatal(err)
		}

		if *Verbose {
			log.Printf("## %s", unit.MakeID(u))
		}

		allRawDeps = append(allRawDeps, rawDeps...)

		for _, rawDep := range rawDeps {
			if *Verbose {
				log.Printf("%+v", rawDep)
			}

			if *resolve {
				log.Printf("# resolves to:")
				resolvedDep, err := dep2.Resolve(rawDep, c, task2.DefaultContext)
				if err != nil {
					log.Fatal(err)
				}
				log.Printf("%+v", resolvedDep)
			}
		}
	}

	if *jsonOutput {
		PrintJSON(allRawDeps, "")
	}
}
