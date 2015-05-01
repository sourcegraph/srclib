package src

import (
	"log"
	"net/url"

	"golang.org/x/net/context"
	"sourcegraph.com/sourcegraph/go-flags"
	"sourcegraph.com/sourcegraph/go-sourcegraph/sourcegraph"
)

func AddClientConfig(p *flags.Parser) {
	p.AddGroup("Client API endpoint", "", &Endpoints)
	p.AddGroup("Client authentication", "", &Credentials)
}

func init() { AddClientConfig(CLI) }

// endpointOpts sets the URLs to use for contacting the Sourcegraph
// server's API.
type endpointOpts struct {
	HTTPEndpoint string `long:"http-endpoint" description:"base URL to HTTP API" default:"http://localhost:3000/api/" env:"SG_API_URL"`
	GRPCEndpoint string `long:"grpc-endpoint" description:"base URL to gRPC API" default:"http://localhost:3100" env:"SG_GRPC_URL"`
}

var Endpoints endpointOpts

// WithEndpoints sets the HTTP and gRPC endpoint in the context.
func (c *endpointOpts) WithEndpoints(ctx context.Context) (context.Context, error) {
	httpEndpoint, err := url.Parse(c.HTTPEndpoint)
	if err != nil {
		return nil, err
	}
	ctx = sourcegraph.WithHTTPEndpoint(ctx, httpEndpoint)

	grpcEndpoint, err := url.Parse(c.GRPCEndpoint)
	if err != nil {
		return nil, err
	}
	ctx = sourcegraph.WithGRPCEndpoint(ctx, grpcEndpoint)

	return ctx, nil
}

// credentialOpts sets the authentication credentials to use when
// contacting the Sourcegraph server's API.
type credentialOpts struct {
	APIKey   string   `long:"api-key" description:"API key" env:"SRC_KEY"`
	Tickets  []string `long:"ticket" description:"tickets" env:"SRCLIB_TICKET"`
	AuthFile string   `long:"auth-file" description:"path to .src-auth" default:"$HOME/.src-auth"`
}

var Credentials credentialOpts

// WithCredentials sets the HTTP and gRPC credentials in the context.
func (c *credentialOpts) WithCredentials(ctx context.Context) (context.Context, error) {
	if c.APIKey != "" {
		ctx = sourcegraph.WithClientCredentials(ctx, &sourcegraph.APIKeyAuth{Key: c.APIKey})
	}
	if len(c.Tickets) != 0 {
		ctx = sourcegraph.WithClientCredentials(ctx, &sourcegraph.TicketAuth{SignedTicketStrings: c.Tickets})
	}
	if c.APIKey == "" && c.AuthFile != "" { // APIKey takes precedence over AuthFile
		userAuth, err := readUserAuth()
		if err != nil {
			return nil, err
		}
		if ua, ok := userAuth[getEndpointURL().String()]; ok {
			ctx = sourcegraph.WithClientCredentials(ctx, &sourcegraph.APIKeyAuth{Key: ua.Key})
		}
	}
	return ctx, nil
}

// Client returns a Sourcegraph API client configured to use the
// specified endpoints and authentication info.
func Client() *sourcegraph.Client {
	ctx := context.Background()
	var err error
	ctx, err = Endpoints.WithEndpoints(ctx)
	if err != nil {
		log.Fatalf("Error constructing API client endpoints: %s.", err)
	}
	ctx, err = Credentials.WithCredentials(ctx)
	if err != nil {
		log.Fatalf("Error constructing API client credentials: %s.", err)
	}
	return sourcegraph.NewClientFromContext(ctx)
}

func getEndpointURL() *url.URL {
	url, err := url.Parse(Endpoints.HTTPEndpoint)
	if err != nil {
		log.Fatal(err, " (in getEndpointURL)")
	}
	return url
}
