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

func upload(args []string) {
	fs := flag.NewFlagSet("upload", flag.ExitOnError)
	r := detectRepository(*dir)
	repoURI := fs.String("repo", string(repo.MakeURI(r.CloneURL)), "repository URI (ex: github.com/alice/foo)")
	commitID := fs.String("commit", r.CommitID, "commit ID (optional)")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: `+Name+` upload [options]

Uploads build data for a repository to Sourcegraph.

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
	localFiles, err := buildstore.ListDataFiles(build.Storage, repo.URI(*repoURI), filepath.Join(*repoURI, *commitID))
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range localFiles {
		uploadFile(file.FullPath, client.BuildDataFileSpec{
			RepositorySpec: client.RepositorySpec{*repoURI},
			CommitID:       file.CommitID,
			Path:           file.Path,
		})
	}
}

func uploadFile(absName string, file client.BuildDataFileSpec) {
	fi, err := build.Storage.Stat(absName)
	if err != nil || !fi.Mode().IsRegular() {
		if *verbose {
			log.Printf("upload: skipping nonexistent file %s", absName)
		}
		return
	}

	kb := float64(fi.Size()) / 1024
	if *verbose {
		log.Printf("Uploading %s (%.1fkb)", absName, kb)
	}

	f, err := build.Storage.Open(absName)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	_, err = apiclient.BuildData.Upload(file, f)
	if err != nil {
		log.Fatal(err)
	}

	if *verbose {
		log.Printf("Uploaded %s (%.1fkb)", absName, kb)
	}
}
