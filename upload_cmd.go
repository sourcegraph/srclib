package srcgraph

import (
	"flag"
	"fmt"
	"log"
	"os"

	"sourcegraph.com/sourcegraph/client"
	"sourcegraph.com/sourcegraph/repo"
	"sourcegraph.com/sourcegraph/srcgraph/buildstore"
)

func upload(args []string) {
	fs := flag.NewFlagSet("upload", flag.ExitOnError)
	r := detectRepository(*dir)
	repoURI := fs.String("repo", string(repo.MakeURI(r.CloneURL)), "repository URI (ex: github.com/alice/foo)")
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

	repoStore, err := buildstore.NewRepositoryStore(r.RootDir)
	if err != nil {
		log.Fatal(err)
	}

	localFiles, err := repoStore.AllDataFiles()
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range localFiles {
		uploadFile(repoStore, file, *repoURI)
	}
}

func uploadFile(repoStore *buildstore.RepositoryStore, file *buildstore.BuildDataFileInfo, repoURI string) {
	path := repoStore.FilePath(file.CommitID, file.Path)

	fileSpec := client.BuildDataFileSpec{
		RepositorySpec: client.RepositorySpec{repoURI},
		CommitID:       file.CommitID,
		Path:           file.Path,
	}

	fi, err := repoStore.Stat(path)
	if err != nil || !fi.Mode().IsRegular() {
		if *Verbose {
			log.Printf("upload: skipping nonexistent file %s", path)
		}
		return
	}

	kb := float64(fi.Size()) / 1024
	if *Verbose {
		log.Printf("Uploading %s (%.1fkb)", path, kb)
	}

	f, err := repoStore.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	_, err = apiclient.BuildData.Upload(fileSpec, f)
	if err != nil {
		log.Fatal(err)
	}

	if *Verbose {
		log.Printf("Uploaded %s (%.1fkb)", path, kb)
	}
}
