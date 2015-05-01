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

// EndpointOpts sets the URLs to use for contacting the Sourcegraph
// server's API.
type EndpointOpts struct {
	HTTPEndpoint string `long:"http-endpoint" description:"base URL to HTTP API" default:"http://localhost:3000/api/" env:"SG_API_URL"`
	GRPCEndpoint string `long:"grpc-endpoint" description:"base URL to gRPC API" default:"http://localhost:3100" env:"SG_GRPC_URL"`
}

var Endpoints EndpointOpts

// WithEndpoints sets the HTTP and gRPC endpoint in the context.
func (c *EndpointOpts) WithEndpoints(ctx context.Context) (context.Context, error) {
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

// CredentialOpts sets the authentication credentials to use when
// contacting the Sourcegraph server's API.
type CredentialOpts struct {
	APIKey   string   `long:"api-key" description:"API key" env:"SRC_KEY"`
	Tickets  []string `long:"ticket" description:"tickets" env:"SRCLIB_TICKET"`
	AuthFile string   `long:"auth-file" description:"path to .src-auth" default:"$HOME/.src-auth"`
}

var Credentials CredentialOpts

// WithCredentials sets the HTTP and gRPC credentials in the context.
func (c *CredentialOpts) WithCredentials(ctx context.Context) (context.Context, error) {
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
	_ = "find_unclosed_clients:ignore"
	return sourcegraph.NewClientFromContext(WithClientContext(context.Background()))
}

// WithClientContext returns a copy of parent with client endpoint and
// auth information added.
func WithClientContext(parent context.Context) context.Context {
	var err error
	ctx, err := Endpoints.WithEndpoints(parent)
	if err != nil {
		log.Fatalf("Error constructing API client endpoints: %s.", err)
	}
	ctx, err = Credentials.WithCredentials(ctx)
	if err != nil {
		log.Fatalf("Error constructing API client credentials: %s.", err)
	}
	return ctx
}

func getEndpointURL() *url.URL {
	url, err := url.Parse(Endpoints.HTTPEndpoint)
	if err != nil {
		log.Fatal(err, " (in getEndpointURL)")
	}
	return url
}
