package srcgraph

import (
	"flag"
	"fmt"
	"log"
	"os"

	"sourcegraph.com/sourcegraph/srcgraph/client"
)

func build_(args []string) {
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	rev := fs.String("rev", "", "VCS revision to build (defaults to latest commit on default branch)")
	queue := fs.Bool("queue", true, "enqueue build to be run")
	import_ := fs.Bool("import", true, "import build data into Sourcegraph app/API when build completes")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: `+Name+` build [options] [REPO-URI ...]

Creates a new build for a repository (and optional VCS revision).

The options are:
`)
		fs.PrintDefaults()
		os.Exit(1)
	}
	fs.Parse(args)

	if fs.NArg() == 0 {
		log.Println("Error: at least one repository must be specified.")
		log.Println()
		fs.Usage()
	}

	for _, uri := range fs.Args() {
		build, _, err := apiclient.Builds.Create(
			client.RepositorySpec{URI: uri, CommitID: *rev},
			client.BuildConfig{Queue: *queue, Import: *import_},
		)
		if err != nil {
			log.Fatalf("Error creating build for %q: %s", uri, err)
		}
		log.Printf("%-30s Build #%d", uri, build.BID)
		if *Verbose {
			PrintJSON(build, "")
			log.Println()
		}
	}
}
