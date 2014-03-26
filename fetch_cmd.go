package srcgraph

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/sourcegraph/rwvfs"

	"sourcegraph.com/sourcegraph/client"
	"sourcegraph.com/sourcegraph/repo"
	"sourcegraph.com/sourcegraph/srcgraph/build"
	"sourcegraph.com/sourcegraph/srcgraph/build/buildstore"
)

func fetch(args []string) {
	fs := flag.NewFlagSet("fetch", flag.ExitOnError)
	r := detectRepository(*dir)
	repoURI := fs.String("repo", string(repo.MakeURI(r.CloneURL)), "repository URI (ex: github.com/alice/foo)")
	commitID := fs.String("commit", r.CommitID, "commit ID (optional)")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: `+Name+` fetch [options]

Fetches build data for a repository from Sourcegraph.

The options are:
`)
		fs.PrintDefaults()
		os.Exit(1)
	}
	fs.Parse(args)

	if fs.NArg() != 0 {
		fs.Usage()
	}

	// TODO!(sqs): this filepath.Join is hacky
	var opt *client.BuildDataListOptions
	if *commitID != "" {
		opt = &client.BuildDataListOptions{CommitID: *commitID}
	}
	localFiles, _, err := apiclient.BuildData.List(client.RepositorySpec{URI: *repoURI}, opt)
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range localFiles {
		fetchFile(file.FullPath, *repoURI, file)
	}
}

func fetchFile(absName string, repoURI string, fi *buildstore.BuildDataFileInfo) {
	fileSpec := client.BuildDataFileSpec{
		RepositorySpec: client.RepositorySpec{repoURI},
		CommitID:       fi.CommitID,
		Path:           fi.Path,
	}

	err := rwvfs.MkdirAll(build.Storage, filepath.Dir(absName))
	if err != nil {
		log.Fatal(err)
	}

	kb := float64(fi.Size) / 1024
	if *verbose {
		log.Printf("Fetching %s (%.1fkb)", absName, kb)
	}

	data, _, err := apiclient.BuildData.Get(fileSpec)
	if err != nil {
		log.Fatal(err)
	}

	if *verbose {
		log.Printf("Fetched %s (%.1fkb)", absName, kb)
	}

	f, err := build.Storage.Create(absName)
	defer f.Close()
	if err != nil {
		log.Fatal(err)
	}
	_, err = f.Write(data)
	if err != nil {
		log.Fatal(err)
	}

	if *verbose {
		log.Printf("Saved %s", absName)
	}
}
