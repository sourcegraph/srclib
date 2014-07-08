package srcgraph

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"sourcegraph.com/sourcegraph/srcgraph/client"
)

func build_(args []string) {
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	rev := fs.String("rev", "", "VCS revision to build (defaults to latest commit on default branch)")
	queue := fs.Bool("queue", true, "enqueue build to be run")
	import_ := fs.Bool("import", true, "import build data into Sourcegraph app/API when build completes")
	useCache := fs.Bool("use-cache", true, "use cached build data (if present)")
	force := fs.Bool("force", true, "force build (even if repository has already been built")
	priority := fs.Int("priority", 0, "build priority")
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
			&client.BuildCreateOptions{
				Force:       *force,
				BuildConfig: client.BuildConfig{Queue: *queue, Import: *import_, UseCache: *useCache, Priority: *priority},
			},
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

func buildQueue(args []string) {
	fs := flag.NewFlagSet("build-queue", flag.ExitOnError)
	perPage := fs.Int("-n", 25, "max builds to show")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: `+Name+` build-queue [options]

Displays the build queue of repositories waiting to be built.

The options are:
`)
		fs.PrintDefaults()
		os.Exit(1)
	}
	fs.Parse(args)

	opt := &client.BuildListOptions{
		Queued:      true,
		ListOptions: client.ListOptions{PerPage: *perPage},
	}
	builds, _, err := apiclient.Builds.List(opt)
	if err != nil {
		log.Fatal(err)
	}

	for _, b := range builds {
		// TODO(sqs): show repository URI, not just ID
		log.Printf("%-35s@ %s #%-5d %s ago", b.RepoURI, b.CommitID, b.BID, time.Since(b.CreatedAt))
		if *Verbose {
			PrintJSON(b, "")
			log.Println()
		}
	}
}
