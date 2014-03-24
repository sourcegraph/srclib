package srcgraph

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/sqs/go-makefile/makefile"
	"sourcegraph.com/sourcegraph/client"
	"sourcegraph.com/sourcegraph/repo"
	"sourcegraph.com/sourcegraph/srcgraph/build"
	"sourcegraph.com/sourcegraph/srcgraph/task2"
)

func upload(args []string) {
	fs := flag.NewFlagSet("upload", flag.ExitOnError)
	r := AddRepositoryFlags(fs)
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

	x := task2.NewRecordedContext()
	repoURI := repo.MakeURI(r.CloneURL)

	rules, err := build.CreateMakefile(r.rootDir, r.CloneURL, r.commitID, x)
	if err != nil {
		log.Fatalf("error creating Makefile: %s", err)
	}

	for _, rule := range rules {
		uploadFile(rule.Target(), repoURI, r.commitID)
	}
}

func uploadFile(target makefile.Target, repoURI repo.URI, commitID string) {
	fi, err := os.Stat(target.Name())
	if err != nil || !fi.Mode().IsRegular() {
		if *verbose {
			log.Printf("upload: skipping nonexistent file %s", target.Name())
		}
		return
	}

	kb := float64(fi.Size()) / 1024
	if *verbose {
		log.Printf("Uploading %s (%.1fkb)", target.Name(), kb)
	}

	f, err := os.Open(target.Name())
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	_, err = apiclient.BuildData.Upload(client.BuildDatumSpec{RepositorySpec: client.RepositorySpec{URI: string(repoURI)}, CommitID: commitID, Name: target.(build.Target).RelName()}, f)
	if err != nil {
		log.Fatal(err)
	}

	if *verbose {
		log.Printf("Uploaded %s (%.1fkb)", target.Name(), kb)
	}
}
