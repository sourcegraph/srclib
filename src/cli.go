package src

import (
	"log"
	"net/http"
	"os"

	"github.com/sourcegraph/httpcache"
	"github.com/sourcegraph/httpcache/diskcache"
	"github.com/sqs/go-flags"
	client "sourcegraph.com/sourcegraph/go-sourcegraph/sourcegraph"
	"sourcegraph.com/sourcegraph/srclib/task2"
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

// TODO(sqs): add base URL flag for apiclient
var (
	httpClient = http.Client{Transport: httpcache.NewTransport(diskcache.New("/tmp/srclib-cache"))}
	apiclient  = client.NewClient(&httpClient)
)

var (
	absDir string
)

func init() {
	var err error
	absDir, err = os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
}

func Main() {
	log.SetFlags(0)
	log.SetPrefix("")
	defer task2.FlushAll()

	if _, err := CLI.Parse(); err != nil {
		os.Exit(1)
	}
}
