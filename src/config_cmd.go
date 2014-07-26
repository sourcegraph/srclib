package src

import (
	"flag"
	"fmt"
	"log"
	"os"
)

func config_(args []string) {
	fs := flag.NewFlagSet("config", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: `+Name+` config [options]

Validates and prints a repository's configuration.

The options are:
`)
		fs.PrintDefaults()
		os.Exit(1)
	}
	fs.Parse(args)

	repoConf, err := OpenAndConfigureRepo(Dir)
	if err != nil {
		log.Fatal(err)
	}

	PrintJSON(repoConf.Config, "")
}
