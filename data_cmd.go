package srcgraph

import (
	"flag"
	"fmt"
	"log"
	"os"

	"sourcegraph.com/sourcegraph/client"
	"sourcegraph.com/sourcegraph/srcgraph/buildstore"
	"sourcegraph.com/sourcegraph/srcgraph/repo"
)

func data(args []string) {
	fs := flag.NewFlagSet("data", flag.ExitOnError)
	r := AddRepositoryFlags(fs)
	repoURI := fs.String("repo", string(repo.MakeURI(r.CloneURL)), "repository URI (ex: github.com/alice/foo)")
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

	remoteFiles, _, err := apiclient.BuildData.List(client.RepositorySpec{URI: *repoURI}, r.CommitID, nil)
	if err != nil {
		log.Fatal(err)
	}

	if *remote {
		log.Println("===================== REMOTE")
		PrintJSON(remoteFiles, "")
		log.Println("============================")
	}

	repoStore, err := buildstore.NewRepositoryStore(r.RootDir)
	if err != nil {
		log.Fatal(err)
	}

	localFiles, err := repoStore.AllDataFiles()
	if err != nil {
		log.Fatal(err)
	}

	if *local {
		log.Println("===================== LOCAL")
		PrintJSON(localFiles, "")
		log.Println("============================")
	}
}
