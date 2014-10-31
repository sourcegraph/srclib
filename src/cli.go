package src

import (
	"log"
	"net/http"
	"net/url"
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

var (
	httpClient = http.Client{Transport: httpcache.NewTransport(diskcache.New("/tmp/srclib-cache"))}
	apiclient  = client.NewClient(&httpClient)
)

func init() {
	// Set the API client's base URL from the SRC_ENDPOINT env var, if set.
	if urlStr := os.Getenv("SRC_ENDPOINT"); urlStr != "" {
		u, err := url.Parse(urlStr)
		if err != nil {
			log.Fatal(err)
		}
		apiclient.BaseURL = u
	}
}

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
