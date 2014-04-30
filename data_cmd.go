package srcgraph

import (
	"flag"
	"fmt"
	"log"
	"os"

	"sourcegraph.com/sourcegraph/srcgraph/buildstore"
	"sourcegraph.com/sourcegraph/srcgraph/client"
	"sourcegraph.com/sourcegraph/srcgraph/task2"
)

func data(args []string) {
	fs := flag.NewFlagSet("data", flag.ExitOnError)
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

	context, err := NewJobContext(*Dir, task2.DefaultContext)
	if err != nil {
		log.Fatal(err)
	}

	remoteFiles, _, err := apiclient.BuildData.List(client.RepositorySpec{URI: string(context.Repo.URI)}, context.CommitID, nil)
	if err != nil {
		log.Fatal(err)
	}

	if *remote {
		log.Println("===================== REMOTE")
		PrintJSON(remoteFiles, "")
		log.Println("============================")
	}

	repoStore, err := buildstore.NewRepositoryStore(context.RepoRootDir)
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
