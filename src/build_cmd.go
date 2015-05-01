package src

import (
	"log"

	"golang.org/x/net/context"

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
	cl := Client()

	rrepo, err := getRemoteRepo(cl)
	if err != nil {
		return err
	}

	repoRev := sourcegraph.RepoRevSpec{RepoSpec: rrepo.RepoSpec(), Rev: c.CommitID, CommitID: c.CommitID}
	build, err := cl.Builds.Create(context.TODO(), &sourcegraph.BuildsCreateOp{RepoRev: repoRev, Opt: &sourcegraph.BuildCreateOptions{
		BuildConfig: sourcegraph.BuildConfig{
			Import:   true,
			Queue:    true,
			Priority: int32(c.Priority),
		},
		Force: true,
	}})

	if err != nil {
		return err
	}
	log.Printf("# Created build #%d", build.BID)

	return nil
}
