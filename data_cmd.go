package srcgraph

import (
	"flag"
	"fmt"
	"log"
	"os"

	"sourcegraph.com/sourcegraph/client"
	"sourcegraph.com/sourcegraph/repo"
)

func data(args []string) {
	fs := flag.NewFlagSet("data", flag.ExitOnError)
	r := detectRepository(*dir)
	repoURI := fs.String("repo", string(repo.MakeURI(r.CloneURL)), "repository URI (ex: github.com/alice/foo)")
	commitID := fs.String("commit", r.CommitID, "commit ID (optional)")
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
	data, _, err := apiclient.BuildData.List(client.RepositorySpec{URI: *repoURI}, opt)
	if err != nil {
		log.Fatal(err)
	}

	PrintJSON(data, "")
}
