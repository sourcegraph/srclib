package src

import (
	"log"
	"path/filepath"

	"github.com/sqs/go-flags"
)

var (
	currentRepo    *Repo
	currentRepoErr error
)

func SetRepoOptDefaults(c *flags.Command) {
	// only try to open the current-dir repo once (we'd get the same result each
	// time, since we never modify it)
	if currentRepo == nil && currentRepoErr == nil {
		currentRepo, currentRepoErr = OpenRepo(".")
	}

	if currentRepo != nil {
		if currentRepo.CloneURL != "" {
			SetOptionDefaultValue(c.Group, "repo", string(currentRepo.URI()))
		}

		subdir, err := filepath.Rel(currentRepo.RootDir, absDir)
		if err != nil {
			log.Fatal(err)
		}
		SetOptionDefaultValue(c.Group, "subdir", subdir)
	}
}
