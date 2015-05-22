package src

import (
	"log"

	"golang.org/x/net/context"

	"sourcegraph.com/sourcegraph/go-sourcegraph/sourcegraph"
)

func init() {
	_, err := CLI.AddCommand("push",
		"upload and import the current commit (to a remote)",
		"The push command uploads and imports the current repository commit's build data to a remote. It is a wrapper around `src build-data upload` and `src remote import-build`.",
		&pushCmd,
	)
	if err != nil {
		log.Fatal(err)
	}
}

type PushCmd struct {
	CommitID string `long:"commit" description:"commit ID of build data to operate on"`
}

var pushCmd PushCmd

func (c *PushCmd) Execute(args []string) error {
	cl := Client()

	rrepo, err := getRemoteRepo(cl)
	if err != nil {
		return err
	}

	commitID := localRepo.CommitID
	if c.CommitID != "" {
		commitID = c.CommitID
		buildDataUploadCmd.CommitID = commitID
		remoteImportBuildCmd.CommitID = commitID
	}

	repoSpec := sourcegraph.RepoSpec{URI: rrepo.URI}
	repoRevSpec := sourcegraph.RepoRevSpec{RepoSpec: repoSpec, Rev: commitID}

	if _, err := cl.Repos.GetCommit(context.TODO(), &repoRevSpec); err != nil {
		return err
	}

	if err := buildDataUploadCmd.Execute(nil); err != nil {
		return err
	}
	if err := remoteImportBuildCmd.Execute(nil); err != nil {
		return err
	}
	return nil
}
