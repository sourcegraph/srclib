package vcsutil

import (
	"os"
	"path/filepath"

	"github.com/sourcegraph/go-vcs/vcs"
	"sourcegraph.com/sourcegraph/srcgraph/repo"
)

func Checkout(cloneURL string, vcsType string, rev string) (dir string, commitID string, err error) {
	dir = filepath.Join("/tmp/sg", string(repo.MakeURI(cloneURL)))

	err = os.MkdirAll(filepath.Dir(dir), 0700)
	if err != nil {
		return "", "", err
	}

	var r vcs.Repository
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		r, err = vcs.Clone(vcsType, cloneURL, dir)
		if err != nil {
			return "", "", err
		}
	} else if err != nil {
		return "", "", nil
	} else {
		r, err = vcs.Open(vcsType, dir)
		if err != nil {
			return "", "", err
		}
	}

	// TODO(new-arch): implement Checkout so that the working tree is at rev
	commitID_, err := r.ResolveRevision(rev)
	if err != nil {
		return "", "", err
	}

	return dir, string(commitID_), nil
}
