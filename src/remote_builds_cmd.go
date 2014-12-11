package src

import (
	"fmt"
	"log"
	"time"

	"sourcegraph.com/sourcegraph/go-sourcegraph/sourcegraph"

	"github.com/sqs/go-flags"
)

func initRemoteBuildCmds(remoteGroup *flags.Command) {
	remoteBuildCmd, err := remoteGroup.AddCommand("build",
		"repository operations",
		"The repo subcommands perform operations on remote repositories.",
		&remoteBuildCmd,
	)
	if err != nil {
		log.Fatal(err)
	}
	if repo := openCurrentRepo(); repo != nil {
		SetOptionDefaultValue(remoteBuildCmd.Group, "commit", repo.CommitID)
	}

	_, err = remoteGroup.AddCommand("builds",
		"repository operations",
		"The repo subcommands perform operations on remote repositories.",
		&remoteBuildsCmd,
	)
	if err != nil {
		log.Fatal(err)
	}
}

type RemoteBuildCmd struct {
	CommitID string `short:"c" long:"commit" description:"commit ID to build" required:"yes"`
	Priority int    `short:"p" long:"priority" description:"build priority" default:"2"`
}

var remoteBuildCmd RemoteBuildCmd

func (c *RemoteBuildCmd) Execute(args []string) error {
	cl := NewAPIClientWithAuthIfPresent()

	build, _, err := cl.Builds.Create(sourcegraph.RepoSpec{URI: remoteCmd.RepoURI}, &sourcegraph.BuildCreateOptions{
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

type RemoteBuildsCmd struct {
	N         int    `short:"n" description:"number of builds to show" default:"5"`
	Rev       string `long:"rev" description:"filter builds by revision or commit ID"`
	Queued    bool   `long:"queued"`
	Succeeded bool   `long:"succeeded"`
	Ended     bool   `long:"ended"`
	Failed    bool   `long:"failed"`
	Sort      string `long:"sort" default:"updated_at"`
	Direction string `long:"dir" default:"desc"`
}

var remoteBuildsCmd RemoteBuildsCmd

func (c *RemoteBuildsCmd) Execute(args []string) error {
	cl := NewAPIClientWithAuthIfPresent()

	opt := &sourcegraph.BuildListByRepoOptions{
		Rev: c.Rev,
		BuildListOptions: sourcegraph.BuildListOptions{
			Queued:      c.Queued,
			Succeeded:   c.Succeeded,
			Ended:       c.Ended,
			Failed:      c.Failed,
			Sort:        c.Sort,
			Direction:   c.Direction,
			ListOptions: sourcegraph.ListOptions{PerPage: c.N},
		},
	}
	builds, _, err := cl.Builds.ListByRepo(sourcegraph.RepoSpec{URI: remoteCmd.RepoURI}, opt)
	if err != nil {
		return err
	}

	for _, b := range builds {
		if b.Success {
			fmt.Printf(green("#% 8d")+" succeeded % 9s ago", b.BID, ago(b.EndedAt.Time))
		} else if b.Failure {
			fmt.Printf(red("#% 8d")+" failed % 9s ago", b.BID, ago(b.EndedAt.Time))
		} else if b.StartedAt.Valid {
			fmt.Printf(cyan("#% 8d")+" started % 9s ago", b.BID, ago(b.StartedAt.Time))
		} else {
			fmt.Printf(gray("#% 8d")+" queued % 9s ago", b.BID, ago(b.CreatedAt))
		}
		fmt.Printf("\t%s\n", b.CommitID)
	}

	return nil
}

func ago(t time.Time) string {
	d := time.Since(t)
	d = (d / time.Second) * time.Second
	return d.String()
}
