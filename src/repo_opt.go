package src

import (
	"log"
	"path/filepath"

	"github.com/sqs/go-flags"
)

func setRepoOptDefaults(c *flags.Command) {
	currentRepo, err := OpenRepo(Dir)
	if err != nil {
		log.Println(err)
		return
	}

	opts := c.Options()
	for _, opt := range opts {
		if opt.LongName == "repo" && opt.ValueName == "URI" {
			opt.Default = []string{string(currentRepo.URI())}
		}
		if opt.LongName == "subdir" && opt.ValueName == "DIR" {
			subdir, err := filepath.Rel(currentRepo.RootDir, absDir)
			if err != nil {
				log.Fatal(err)
			}
			opt.Default = []string{subdir}
		}
	}
}
