package srcgraph

import (
	"flag"
	"fmt"
	"log"
	"os"

	"sourcegraph.com/sourcegraph/srcgraph/client"
	"sourcegraph.com/sourcegraph/srcgraph/repo"
)

func repoCreate(args []string) {
	fs := flag.NewFlagSet("repo-create", flag.ExitOnError)
	vcsType := fs.String("vcs", "git", "VCS type ('git' or 'hg')")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: `+Name+` repo-create [-vcs=(git|hg)] CLONE-URL

Creates new repositories with the specified clone URLs and VCS type.
Repository URIs (e.g., github.com/user/repo) may be specified instead
of full clone URLs (e.g., git://github.com/user/repo.git).

The options are:
`)
		fs.PrintDefaults()
		os.Exit(1)
	}
	fs.Parse(args)

	if fs.NArg() == 0 {
		log.Println("Error: at least one repository clone URL must be specified.")
		log.Println()
		fs.Usage()
	}

	for _, urlStr := range fs.Args() {
		repo, _, err := apiclient.Repositories.Create(client.NewRepositorySpec{Type: repo.VCS(*vcsType), CloneURLStr: urlStr})
		if err != nil {
			log.Fatalf("Error creating repository with %q: %s", urlStr, err)
		}
		log.Printf("%-45s Repository #%d", repo.URI, repo.RID)
		if *Verbose {
			PrintJSON(repo, "")
			log.Println()
		}
	}
}
