package vcsutil

import (
	"os"
	"path/filepath"

	"github.com/sourcegraph/go-vcs"
	"sourcegraph.com/sourcegraph/srcgraph/repo"
)

func Checkout(cloneURL string, vcsType vcs.VCS, rev string) (string, string, error) {
	dir := filepath.Join("/tmp/sg", string(repo.MakeURI(cloneURL)))

	err := os.MkdirAll(filepath.Dir(dir), 0700)
	if err != nil {
		return "", "", err
	}

	r, err := vcs.CloneOrOpen(vcsType, cloneURL, dir)
	if err != nil {
		return "", "", err
	}

	err = r.Download()
	if err != nil {
		return "", "", err
	}

	if rev != "" {
		_, err = r.CheckOut(rev)
		if err != nil {
			return "", "", err
		}
	}

	commitID, err := r.CurrentCommitID()
	if err != nil {
		return "", "", err
	}

	return dir, commitID, nil
}
