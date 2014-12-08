package src

import (
	"log"
	"net/http"
	"net/url"
	"os"

	"strconv"

	"github.com/gregjones/httpcache"
	"github.com/gregjones/httpcache/diskcache"
	"github.com/sqs/go-flags"
	"sourcegraph.com/sourcegraph/go-sourcegraph/auth"
	"sourcegraph.com/sourcegraph/go-sourcegraph/sourcegraph"
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
	// endpointURL is the API endpoint used as apiclient.BaseURL. It
	// is set from SRC_ENDPOINT and it defaults to hitting
	// Sourcegraph.
	endpointURL = getEndpointURL()

	// apiclient is the API client, created using
	// newAPIClientWithAuthIfPresent. It is authenticated if the user
	// has previously stored login credentials for the current
	// endpointURL.
	apiclient = newAPIClientWithAuthIfPresent()
)

func getEndpointURL() *url.URL {
	if urlStr := os.Getenv("SRC_ENDPOINT"); urlStr != "" {
		u, err := url.Parse(urlStr)
		if err != nil {
			log.Fatal("Parsing SRC_ENDPOINT URL string:", err)
		}
		return u
	}
	return &url.URL{Scheme: "https", Host: "sourcegraph.com", Path: "/api/"}
}

// newAPIClient creates a new Sourcegraph API client for the endpoint
// given by endpointURL (a global) and that is authenticated using the
// credentials in ua (if non-nil).
func newAPIClient(ua *userEndpointAuth) *sourcegraph.Client {
	cache := httpcache.NewTransport(diskcache.New("/tmp/srclib-cache"))
	var httpClient http.Client
	if ua == nil {
		// Unauthenticated API client.
		if GlobalOpt.Verbose {
			log.Printf("# Using unauthenticated API client for endpoint %s.", endpointURL)
		}
		httpClient.Transport = cache
	} else {
		// Authenticated API client.
		if GlobalOpt.Verbose {
			log.Printf("# Using authenticated API client for endpoint %s (UID %d).", endpointURL, ua.UID)
		}
		httpClient.Transport = &auth.BasicAuthTransport{Username: strconv.Itoa(ua.UID), Password: ua.Key, Transport: cache}
	}
	c := sourcegraph.NewClient(&httpClient)
	c.BaseURL = endpointURL
	return c
}

// newAPIClientWithAuthIfPresent calls newAPIClient with the user auth
// credentials from the userAuthFile (if present), and otherwise
// creates an unauthed API client.
func newAPIClientWithAuthIfPresent() *sourcegraph.Client {
	a, err := readUserAuth()
	if err != nil {
		log.Fatal("Reading user auth:", err)
	}
	ua := a[endpointURL.String()]
	return newAPIClient(ua)
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
