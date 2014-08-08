// +build off

package src

import (
	"flag"
	"fmt"
	"log"
	"os"

	"sourcegraph.com/sourcegraph/srclib/grapher"
)

func graph_(args []string) {
	fs := flag.NewFlagSet("graph", flag.ExitOnError)
	jsonOutput := fs.Bool("json", false, "show JSON output")
	summary := fs.Bool("summary", true, "summarize output data")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: `+Name+` graph [options] [unit...]

Analyze a repository's source code for definitions and references. If unit(s)
are specified, only source units with matching IDs will be graphed.

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

	for _, u := range repoConf.Config.SourceUnits {
		if !SourceUnitMatchesArgs(sourceUnitSpecs, u) {
			continue
		}

		output, err := grapher.Graph(repoConf.RootDir, u, repoConf.Config)
		if err != nil {
			log.Fatal(err)
		}

		if *summary || GlobalOpt.Verbose {
			log.Printf("%s output summary:", u.ID())
			log.Printf(" - %d defs", len(output.Defs))
			log.Printf(" - %d refs", len(output.Refs))
			log.Printf(" - %d docs", len(output.Docs))
		}

		if *jsonOutput {
			PrintJSON(output, "")
		}

		fmt.Println()
	}
}
