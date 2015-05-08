package src

import (
	"fmt"
	"log"
)

// Version of sgx.
//
// For releases, this is set using the -X flag to `go tool ld`. See
// http://stackoverflow.com/a/11355611.
var Version = "dev"

func init() {
	_, err := CLI.AddCommand("version",
		"show version",
		"The version subcommand displays the current version of this src program.",
		&versionCmd{},
	)
	if err != nil {
		log.Fatal(err)
	}
}

type versionCmd struct{}

func (v *versionCmd) Execute(_ []string) error {
	fmt.Println(Version)
	return nil
}
