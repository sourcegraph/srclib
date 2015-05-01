package src

import (
	"log"

	"sourcegraph.com/sourcegraph/go-flags"
)

var CLI = flags.NewNamedParser("src", flags.Default)

// GlobalOpt contains global options.
var GlobalOpt struct {
	Verbose bool `short:"v" description:"show verbose output"`
}

func init() {
	CLI.LongDescription = "src builds projects, analyzes source code, and queries Sourcegraph."
	CLI.AddGroup("Global options", "", &GlobalOpt)
}

func Main() error {
	log.SetFlags(0)
	log.SetPrefix("")

	_, err := CLI.Parse()
	return err
}
