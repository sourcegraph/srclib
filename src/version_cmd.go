package src

import (
	"fmt"
	"log"

	"github.com/inconshreveable/go-update/check"
)

func init() {
	_, err := CLI.AddCommand("version",
		"show version",
		"Show version.",
		&versionCmd,
	)
	if err != nil {
		log.Fatal(err)
	}
}

type VersionCmd struct{}

var versionCmd VersionCmd

func (c *VersionCmd) Execute(args []string) error {
	fmt.Printf("srclib v%s\n", Version)

	r, err := checkForUpdate()
	if err == check.NoUpdateAvailable {
		log.Println("\nYou are on the latest version of src.")
		return nil
	} else if err != nil {
		return err
	}

	if r != nil {
		log.Printf("\nA newer version of src is available: v%s.", r.Version)
		log.Println("Run 'src selfupdate' to update.")
	}

	return nil
}
