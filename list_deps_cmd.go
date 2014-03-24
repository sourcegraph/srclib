package srcgraph

import (
	"flag"
	"fmt"
	"log"
	"os"

	"sourcegraph.com/sourcegraph/repo"
	"sourcegraph.com/sourcegraph/srcgraph/dep2"
	"sourcegraph.com/sourcegraph/srcgraph/scan"
	"sourcegraph.com/sourcegraph/srcgraph/task2"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
)

func listDeps(args []string) {
	fs := flag.NewFlagSet("list-deps", flag.ExitOnError)
	resolve := fs.Bool("resolve", false, "resolve deps and print resolutions")
	jsonOutput := fs.Bool("json", false, "show JSON output")
	r := AddRepositoryFlags(fs)
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
	repoURI := repo.MakeURI(r.CloneURL)

	x := task2.DefaultContext
	c, err := scan.ReadDirConfigAndScan(r.RootDir, repoURI, x)
	if err != nil {
		log.Fatal(err)
	}

	allRawDeps := []*dep2.RawDependency{}
	for _, u := range c.SourceUnits {
		if !SourceUnitMatchesArgs(sourceUnitSpecs, u) {
			continue
		}

		rawDeps, err := dep2.List(r.RootDir, u, c, x)
		if err != nil {
			log.Fatal(err)
		}

		if *verbose {
			log.Printf("## %s", unit.MakeID(u))
		}

		allRawDeps = append(allRawDeps, rawDeps...)

		for _, rawDep := range rawDeps {
			if *verbose {
				log.Printf("%+v", rawDep)
			}

			if *resolve {
				log.Printf("# resolves to:")
				resolvedDep, err := dep2.Resolve(rawDep, c, x)
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
