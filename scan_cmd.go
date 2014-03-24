package srcgraph

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"sourcegraph.com/sourcegraph/repo"
	"sourcegraph.com/sourcegraph/srcgraph/scan"
	"sourcegraph.com/sourcegraph/srcgraph/task2"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
)

func scan_(args []string) {
	fs := flag.NewFlagSet("scan", flag.ExitOnError)
	r := AddRepositoryFlags(fs)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: `+Name+` scan [options]

Scans a repository for source units.

The options are:
`)
		fs.PrintDefaults()
		os.Exit(1)
	}
	fs.Parse(args)

	x := task2.DefaultContext

	c, err := scan.ReadDirConfigAndScan(r.RootDir, repo.MakeURI(r.CloneURL), x)
	if err != nil {
		log.Fatal(err)
	}

	for _, u := range c.SourceUnits {
		fmt.Printf("## %s\n", unit.MakeID(u))
		for _, p := range u.Paths() {
			fmt.Printf("  %s\n", p)
		}
		if *verbose {
			jsonStr, err := json.MarshalIndent(u, "\t", "  ")
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(string(jsonStr))
		}
	}
}
