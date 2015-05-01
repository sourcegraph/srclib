package src

import (
	"fmt"

	"golang.org/x/net/context"

	"sourcegraph.com/sourcegraph/go-sourcegraph/sourcegraph"
)

// getRemoteRepo gets the remote repository that corresponds to the
// local repository (from openLocalRepo). It does not respect any
// flags that override the repo URI to use. Commands that need to
// allow the user to override the repo URI should be under the
// "remote" subcommand and use "RemoteCmd.getRemoteRepo".
func getRemoteRepo(cl *sourcegraph.Client) (*sourcegraph.Repo, error) {
	lrepo, err := openLocalRepo()
	if err != nil {
		return nil, err
	}
	if lrepo.CloneURL == "" {
		return nil, errNoVCSCloneURL
	}
	uri := lrepo.URI()
	if uri == "" {
		return nil, fmt.Errorf("getRemoteRepo: the local repo's URI is malformed: %s", lrepo.CloneURL)
	}

	rrepo, err := cl.Repos.Get(context.TODO(), &sourcegraph.RepoSpec{URI: uri})
	if err != nil {
		return nil, fmt.Errorf("repo %s: %s", uri, err)
	}
	return rrepo, nil
}
