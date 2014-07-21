package src

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
)

func scan_(args []string) {
	fs := flag.NewFlagSet("scan", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: `+Name+` scan [options]

Scans a repository for source units.

The options are:
`)
		fs.PrintDefaults()
		os.Exit(1)
	}
	fs.Parse(args)

	repoConf, err := OpenAndConfigureRepo(*Dir)
	if err != nil {
		log.Fatal(err)
	}

	for _, u := range repoConf.Config.SourceUnits {
		fmt.Printf("## %s\n", u.ID())
		if *Verbose {
			jsonStr, err := json.MarshalIndent(u, "\t", "  ")
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(string(jsonStr))
		}
	}
}
