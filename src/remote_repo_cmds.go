package src

import (
	"fmt"
	"log"

	"sourcegraph.com/sourcegraph/go-sourcegraph/sourcegraph"

	"github.com/sqs/go-flags"
)

func initRemoteRepoCmds(remoteGroup *flags.Command) {
	_, err := remoteGroup.AddCommand("repo",
		"repository",
		"The repo subcommands perform operations on remote repositories.",
		&remoteRepoCmd,
	)
	if err != nil {
		log.Fatal(err)
	}
}

type RemoteRepoCmd struct{}

var remoteRepoCmd RemoteRepoCmd

func (c *RemoteRepoCmd) Execute(args []string) error {
	cl := NewAPIClientWithAuthIfPresent()

	remoteRepo, _, err := cl.Repos.Get(sourcegraph.RepoSpec{URI: remoteCmd.RepoURI}, nil)
	if err != nil {
		return err
	}

	fmt.Printf("%s\n", bold(remoteRepo.URI))
	if remoteRepo.URIAlias != "" {
		fmt.Printf(fade("%s (alias)\n"), remoteRepo.URIAlias)
	}
	fmt.Println()

	if remoteRepo.Description != "" {
		fmt.Printf("%s\n", remoteRepo.Description)
		fmt.Println()
	}

	if remoteRepo.HomepageURL != "" {
		fmt.Printf("Homepage:       %s\n", remoteRepo.HomepageURL)
	}
	if remoteRepo.HTTPCloneURL != "" {
		fmt.Printf("Clone (HTTP):   %s (%s)\n", remoteRepo.HTTPCloneURL, remoteRepo.VCS)
	}
	if remoteRepo.SSHCloneURL != "" {
		fmt.Printf("Clone (SSH):    %s (%s)\n", remoteRepo.SSHCloneURL, remoteRepo.VCS)
	}
	fmt.Printf("Default branch: %s\n", remoteRepo.DefaultBranch)
	if remoteRepo.Language != "" {
		fmt.Printf("Language:       %s\n", remoteRepo.Language)
	}

	log.Println()
	log.Println()
	log.Printf("# View recent builds: src remote builds")
	log.Printf("# Trigger new build:  src remote build")
	log.Println()

	return nil
}
