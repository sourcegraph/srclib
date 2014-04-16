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
	"sourcegraph.com/sourcegraph/srcgraph/buildstore"
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

	localFiles, _, err := apiclient.BuildData.List(client.RepositorySpec{URI: *repoURI}, *commitID, nil)
	if err != nil {
		log.Fatal(err)
	}

	repoStore, err := buildstore.NewRepositoryStore(r.RootDir)
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range localFiles {
		fetchFile(repoStore, *repoURI, file)
	}
}

func fetchFile(repoStore *buildstore.RepositoryStore, repoURI string, fi *buildstore.BuildDataFileInfo) {
	path := repoStore.FilePath(fi.CommitID, fi.Path)

	fileSpec := client.BuildDataFileSpec{
		Repo:     client.RepositorySpec{repoURI},
		CommitID: fi.CommitID,
		Path:     fi.Path,
	}

	kb := float64(fi.Size) / 1024
	if *Verbose {
		log.Printf("Fetching %s (%.1fkb)", path, kb)
	}

	data, _, err := apiclient.BuildData.Get(fileSpec)
	if err != nil {
		log.Fatal(err)
	}

	if *Verbose {
		log.Printf("Fetched %s (%.1fkb)", path, kb)
	}

	err = rwvfs.MkdirAll(repoStore, filepath.Dir(path))
	if err != nil {
		log.Fatal(err)
	}

	f, err := repoStore.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	_, err = f.Write(data)
	if err != nil {
		log.Fatal(err)
	}

	if *Verbose {
		log.Printf("Saved %s", path)
	}
}
