package cli

import (
	"errors"
	"fmt"

	"sourcegraph.com/sourcegraph/rwvfs"
	"sourcegraph.com/sourcegraph/srclib/buildstore"
)

// GetBuildDataFS gets the build data file system for repo at
// commitID. If local is true, repo is ignored and build data is
// fetched for the local repo.
var GetBuildDataFS = func(local bool, repo, commitID string) (rwvfs.FileSystem, string, error) {
	if !local {
		return nil, "", errors.New("remote build data is unsupported in srclib (requires Sourcegraph)")
	}
	return GetLocalBuildDataFS(commitID)
}

func GetLocalBuildDataFS(commitID string) (rwvfs.FileSystem, string, error) {
	lrepo, err := OpenLocalRepo()
	if lrepo == nil || lrepo.RootDir == "" || commitID == "" {
		return nil, "", err
	}
	localStore, err := buildstore.LocalRepo(lrepo.RootDir)
	if err != nil {
		return nil, "", err
	}
	return localStore.Commit(commitID), fmt.Sprintf("local repository (root dir %s, commit %s)", lrepo.RootDir, commitID), nil
}
