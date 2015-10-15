package cli

import (
	"log"

	"github.com/alexsaveliev/go-colorable-wrapper/fmtc"
)

func init() {
	_, err := CLI.AddCommand("repo",
		"display current repo info",
		"The repo subcommand displays autodetected info about the current repo.",
		&repoCmd{},
	)
	if err != nil {
		log.Fatal(err)
	}
}

type repoCmd struct{}

func (c *repoCmd) Execute(args []string) error {
	repo, err := OpenRepo(".")
	if err != nil {
		return err
	}
	fmtc.Println("# Current repository:")
	fmtc.Println("URI:", repo.URI())
	fmtc.Println("Clone URL:", repo.CloneURL)
	fmtc.Println("VCS:", repo.VCSType)
	fmtc.Println("Root dir:", repo.RootDir)
	fmtc.Println("Commit ID:", repo.CommitID)
	return nil
}
