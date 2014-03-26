package srcgraph

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"sourcegraph.com/sourcegraph/client"
	"sourcegraph.com/sourcegraph/repo"
	"sourcegraph.com/sourcegraph/srcgraph/build"
	"sourcegraph.com/sourcegraph/srcgraph/build/buildstore"
)

func data(args []string) {
	fs := flag.NewFlagSet("data", flag.ExitOnError)
	r := detectRepository(*dir)
	repoURI := fs.String("repo", string(repo.MakeURI(r.CloneURL)), "repository URI (ex: github.com/alice/foo)")
	commitID := fs.String("commit", r.CommitID, "commit ID (optional)")
	remote := fs.Bool("remote", true, "show remote data")
	local := fs.Bool("local", true, "show local data")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: `+Name+` data [options]

Lists available repository data.

The options are:
`)
		fs.PrintDefaults()
		os.Exit(1)
	}
	fs.Parse(args)

	if fs.NArg() != 0 {
		fs.Usage()
	}

	var opt *client.BuildDataListOptions
	if *commitID != "" {
		opt = &client.BuildDataListOptions{CommitID: *commitID}
	}
	remoteData, _, err := apiclient.BuildData.List(client.RepositorySpec{URI: *repoURI}, opt)
	if err != nil {
		log.Fatal(err)
	}

	if *remote {
		log.Println("===================== REMOTE")
		PrintJSON(remoteData, "")
		log.Println("============================")
	}

	// TODO!(sqs): this filepath.Join is hacky
	localData, err := buildstore.ListDataFiles(build.Storage, repo.URI(*repoURI), filepath.Join(*repoURI, *commitID))
	if err != nil {
		log.Fatal(err)
	}
	if *local {
		log.Println("===================== LOCAL")
		PrintJSON(localData, "")
		log.Println("============================")
	}
}
