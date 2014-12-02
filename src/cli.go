package src

import (
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/gregjones/httpcache"
	"github.com/gregjones/httpcache/diskcache"
	"github.com/sqs/go-flags"
	"sourcegraph.com/sourcegraph/go-sourcegraph/auth"
	client "sourcegraph.com/sourcegraph/go-sourcegraph/sourcegraph"
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
	httpClient http.Client
	apiclient  *client.Client
)

func init() {
	// Initialize API client, setting auth data from SRC_USER_ID and SRC_USER_KEY if those env vars are set.
	unauthedTransport := httpcache.NewTransport(diskcache.New("/tmp/srclib-cache"))
	uid, uKey := os.Getenv("SRC_USER_ID"), os.Getenv("SRC_USER_KEY")
	if uid == "" {
		httpClient = http.Client{Transport: unauthedTransport}
	} else {
		authedTransport := &auth.BasicAuthTransport{Username: uid, Password: uKey, Transport: unauthedTransport}
		httpClient = http.Client{Transport: authedTransport}
	}
	apiclient = client.NewClient(&httpClient)

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

func Main() error {
	log.SetFlags(0)
	log.SetPrefix("")

	_, err := CLI.Parse()
	return err
}
