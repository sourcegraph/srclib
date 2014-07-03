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

func repoRefreshProfile(args []string) {
	fs := flag.NewFlagSet("repo-refresh-profile", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: `+Name+` repo-refresh-profile [REPO-URI ...]

Triggers a refresh of the profiles of repositories whose URIs are specified as
arguments.

The options are:
`)
		fs.PrintDefaults()
		os.Exit(1)
	}
	fs.Parse(args)

	if fs.NArg() == 0 {
		fs.Usage()
	}

	for _, uri := range fs.Args() {
		_, err := apiclient.Repositories.RefreshProfile(client.RepositorySpec{URI: uri})
		if err != nil {
			log.Fatalf("Error triggering a refresh of repository profile for %q: %s.", uri, err)
		}
	}
}

func repoRefreshVCSData(args []string) {
	fs := flag.NewFlagSet("repo-refresh-vcs-data", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: `+Name+` repo-refresh-vcs-data [REPO-URI ...]

Triggers a refresh of the VCS (git/hg/etc.) data of repositories whose URIs are
specified as arguments.

The options are:
`)
		fs.PrintDefaults()
		os.Exit(1)
	}
	fs.Parse(args)

	if fs.NArg() == 0 {
		fs.Usage()
	}

	for _, uri := range fs.Args() {
		_, err := apiclient.Repositories.RefreshVCSData(client.RepositorySpec{URI: uri})
		if err != nil {
			log.Fatalf("Error triggering a refresh of repository VCS data for %q: %s.", uri, err)
		}
	}
}

func repoComputeStats(args []string) {
	fs := flag.NewFlagSet("repo-compute-stats", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: `+Name+` repo-compute-stats [REPO-URI ...]

Triggers an update of statistics for the repositories whose URIs are specified
as arguments.

The options are:
`)
		fs.PrintDefaults()
		os.Exit(1)
	}
	fs.Parse(args)

	if fs.NArg() == 0 {
		fs.Usage()
	}

	for _, uri := range fs.Args() {
		repoSpec := client.RepositorySpec{URI: uri}
		repo, _, err := apiclient.Repositories.Get(repoSpec, &client.RepositoryGetOptions{ResolveRevision: true})
		if err != nil {
			log.Fatalf("Error resolving revision %q for repository %q: %s.", repoSpec.CommitID, repoSpec.URI, err)
		}

		repoSpec.CommitID = repo.CommitID
		_, err = apiclient.Repositories.ComputeStats(repoSpec)
		if err != nil {
			log.Fatalf("Error triggering a computation of repository stats for %q: %s.", repoSpec.URI, err)
		}
	}
}
