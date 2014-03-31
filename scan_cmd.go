package srcgraph

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sourcegraph.com/sourcegraph/srcgraph/task2"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
)

func scan_(args []string) {
	fs := flag.NewFlagSet("scan", flag.ExitOnError)
	r := AddRepositoryFlags(fs)
	rc := AddRepositoryConfigFlags(fs, r)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: `+Name+` scan [options]

Scans a repository for source units.

The options are:
`)
		fs.PrintDefaults()
		os.Exit(1)
	}
	fs.Parse(args)

	c := rc.GetRepositoryConfig(task2.DefaultContext)

	for _, u := range c.SourceUnits {
		fmt.Printf("## %s\n", unit.MakeID(u))
		for _, p := range u.Paths() {
			fmt.Printf("  %s\n", p)
		}
		if *Verbose {
			jsonStr, err := json.MarshalIndent(u, "\t", "  ")
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(string(jsonStr))
		}
	}
}
