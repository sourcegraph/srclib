package src

import (
	"log"
	"net/http"
	"net/url"
	"os"

	"strconv"

	"github.com/sourcegraph/httpcache"
	"github.com/sourcegraph/httpcache/diskcache"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"sourcegraph.com/sourcegraph/go-flags"
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

// getPermGrantTickets gets signed perm grant ticket strings from the
// SRCLIB_TICKET env var. Currently only 1 ticket can be provided. The
// signed ticket string should not contain the "Sourcegraph-Ticket "
// prefix.
func getPermGrantTickets() []string {
	tkstr := os.Getenv("SRCLIB_TICKET")
	if tkstr == "" {
		return nil
	}
	return []string{tkstr}
}

// newAPIClient creates a new Sourcegraph API client for the endpoint
// given by endpointURL (a global) and that is authenticated using the
// credentials in ua (if non-nil) and perm grant ticket (if one is
// provided in SRCLIB_TICKET).
func newAPIClient(ua *userEndpointAuth, cache bool) *sourcegraph.Client {
	endpointURL := getEndpointURL()

	var transport http.RoundTripper
	opts := []grpc.DialOption{
		grpc.WithCodec(sourcegraph.GRPCCodec),
	}
	if endpointURL.Scheme == "https" {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")))
	}

	if cache {
		transport = http.RoundTripper(httpcache.NewTransport(diskcache.New("/tmp/srclib-cache")))
	} else {
		transport = http.DefaultTransport
	}

	if tickets := getPermGrantTickets(); len(tickets) > 0 {
		log.Println("# Using perm grant ticket from SRCLIB_TICKET.")
		cred := &sourcegraph.TicketAuth{SignedTicketStrings: tickets, Transport: transport}
		opts = append(opts, grpc.WithPerRPCCredentials(cred))
		transport = cred
	}

	if ua == nil {
		// Unauthenticated API client.
		if GlobalOpt.Verbose {
			log.Printf("# Using unauthenticated API client for endpoint %s.", endpointURL)
		}
	} else {
		// Authenticated API client.
		if GlobalOpt.Verbose {
			log.Printf("# Using authenticated API client for endpoint %s (UID %d).", endpointURL, ua.UID)
		}
		cred := &sourcegraph.KeyAuth{UID: ua.UID, Key: ua.Key, Transport: transport}
		opts = append(opts, grpc.WithPerRPCCredentials(cred))
		transport = cred
	}

	//c := sourcegraph.NewClient(&http.Client{Transport: transport})
	//c.BaseURL = endpointURL

	conn, err := grpc.Dial("localhost:3100", opts...)
	if err != nil {
		panic(err)
	}
	c := sourcegraph.NewClient(&http.Client{Transport: transport}, conn)
	return c
}

// NewAPIClientWithAuthIfPresent calls newAPIClient with the user auth
// credentials from the userAuthFile (if present), and otherwise
// creates an unauthed API client.
func NewAPIClientWithAuthIfPresent() *sourcegraph.Client {
	return newAPIClientWithAuth(true)
}

func newAPIClientWithAuth(cache bool) *sourcegraph.Client {
	var ua *userEndpointAuth
	if uidStr, key := os.Getenv("SRC_UID"), os.Getenv("SRC_KEY"); uidStr != "" && key != "" {
		uid, err := strconv.Atoi(uidStr)
		if err != nil {
			log.Fatal("Parsing SRC_UID:", err)
		}
		ua = &userEndpointAuth{UID: uid, Key: key}
	} else {
		a, err := readUserAuth()
		if err != nil {
			log.Fatal("Reading user auth:", err)
		}
		ua = a[getEndpointURL().String()]
	}
	return newAPIClient(ua, cache)
}

func Main() error {
	log.SetFlags(0)
	log.SetPrefix("")

	_, err := CLI.Parse()
	return err
}
