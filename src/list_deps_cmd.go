// +build off

package src

import (
	"flag"
	"fmt"
	"log"
	"os"

	"sourcegraph.com/sourcegraph/srclib/dep"
)

func listDeps(args []string) {
	fs := flag.NewFlagSet("list-deps", flag.ExitOnError)
	resolve := fs.Bool("resolve", false, "resolve deps and print resolutions")
	jsonOutput := fs.Bool("json", false, "show JSON output")
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

	repoConf, err := OpenAndConfigureRepo(Dir)
	if err != nil {
		log.Fatal(err)
	}

	allRawDeps := []*dep.RawDependency{}
	for _, u := range repoConf.Config.SourceUnits {
		if !SourceUnitMatchesArgs(sourceUnitSpecs, u) {
			continue
		}

		rawDeps, err := dep.List(repoConf.RootDir, u, repoConf.Config)
		if err != nil {
			log.Fatal(err)
		}

		if GlobalOpt.Verbose {
			log.Printf("%s", u.ID())
		}

		allRawDeps = append(allRawDeps, rawDeps...)

		for _, rawDep := range rawDeps {
			if GlobalOpt.Verbose {
				log.Printf("%+v", rawDep)
			}

			if *resolve {
				log.Printf("# resolves to:")
				resolvedDep, err := dep.Resolve(rawDep, repoConf.Config)
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
