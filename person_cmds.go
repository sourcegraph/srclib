package srcgraph

import (
	"flag"
	"fmt"
	"log"
	"os"

	"sourcegraph.com/sourcegraph/srcgraph/client"
)

const PersonSpecHelp = `Specify UIDs as '$n' (use \$n to shell-escape), as in '$123' for UID 123.`

func personRefreshProfile(args []string) {
	fs := flag.NewFlagSet("person-refresh-profile", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: `+Name+` person-refresh-profile [PERSON-SPEC ...]

Triggers a refresh of the profiles of people specified as arguments.

The options are:
`)
		fs.PrintDefaults()
		os.Exit(1)
	}
	fs.Parse(args)

	if fs.NArg() == 0 {
		fs.Usage()
	}

	for _, specStr := range fs.Args() {
		ps, err := client.ParsePersonSpec(specStr)
		if err != nil {
			log.Fatalf("Error parsing person specifier %q: %s.", specStr, err)
		}

		_, err = apiclient.People.RefreshProfile(ps)
		if err != nil {
			log.Fatalf("Error triggering a refresh of person profile for %v: %s.", ps, err)
		}
	}
}

func personComputeStats(args []string) {
	fs := flag.NewFlagSet("person-compute-stats", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: `+Name+` person-compute-stats [PERSON-SPEC ...]

Triggers an update of statistics for the people specified as arguments.

The options are:
`)
		fs.PrintDefaults()
		os.Exit(1)
	}
	fs.Parse(args)

	if fs.NArg() == 0 {
		fs.Usage()
	}

	for _, specStr := range fs.Args() {
		ps, err := client.ParsePersonSpec(specStr)
		if err != nil {
			log.Fatalf("Error parsing person specifier %q: %s.", specStr, err)
		}

		_, err = apiclient.People.ComputeStats(ps)
		if err != nil {
			log.Fatalf("Error triggering a computation of person stats for %v: %s.", ps, err)
		}
	}
}
