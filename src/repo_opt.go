package src

import (
	"log"
	"path/filepath"

	"github.com/sqs/go-flags"
)

func SetRepoOptDefaults(c *flags.Command) {
	currentRepo, err := OpenRepo(".")
	if err != nil {
		log.Println(err)
		return
	}

	SetOptionDefaultValue(c.Group, "repo", string(currentRepo.URI()))

	subdir, err := filepath.Rel(currentRepo.RootDir, absDir)
	if err != nil {
		log.Fatal(err)
	}
	SetOptionDefaultValue(c.Group, "subdir", subdir)
}
