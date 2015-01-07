package src

import (
	"log"

	"sourcegraph.com/sourcegraph/go-sourcegraph/sourcegraph"
)

func init() {
	buildCmd, err := CLI.AddCommand("build",
		"trigger a remote build",
		"The build command triggers a remote build of the repository.",
		&buildCmd,
	)
	if err != nil {
		log.Fatal(err)
	}
	setDefaultCommitIDOpt(buildCmd)
}

type BuildCmd struct {
	CommitID string `short:"c" long:"commit" description:"commit ID to build" required:"yes"`
	Priority int    `short:"p" long:"priority" description:"build priority" default:"2"`
}

var buildCmd BuildCmd

func (c *BuildCmd) Execute(args []string) error {
	cl := NewAPIClientWithAuthIfPresent()

	rrepo, err := getRemoteRepo(cl)
	if err != nil {
		return err
	}

	build, _, err := cl.Builds.Create(rrepo.RepoSpec(), &sourcegraph.BuildCreateOptions{
		BuildConfig: sourcegraph.BuildConfig{
			Import:   true,
			Queue:    true,
			Priority: c.Priority,
			CommitID: c.CommitID,
		},
		Force: true,
	})
	if err != nil {
		return err
	}
	log.Printf("# Created build #%d", build.BID)

	return nil
}
