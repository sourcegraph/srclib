package toolchain

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsouza/go-dockerclient"
)

// newTLSDockerClient creates a new TLS docker client using the certificates
// from the the given docker certificate path (i.e. $DOCKER_CERT_PATH)
// connecting to the given docker host (i.e. $DOCKER_HOST).
//
// This should only be used when $DOCKER_CERT_PATH is set and is not an empty
// string.
//
// See https://github.com/fsouza/go-dockerclient/issues/166 for details on why
// this function is needed at all.
func newTLSDockerClient(certPath, host string) (*docker.Client, error) {
	// Enforce the host URL scheme to be HTTPS.
	h, err := url.Parse(host)
	if err != nil {
		return nil, err
	}
	h.Scheme = "https"

	// Create certificate pool.
	roots := x509.NewCertPool()

	// Load client authority certificate.
	pemData, err := ioutil.ReadFile(filepath.Join(certPath, "ca.pem"))
	if err != nil {
		return nil, err
	}
	roots.AppendCertsFromPEM(pemData)

	// Create certificate.
	cert, err := tls.LoadX509KeyPair(filepath.Join(certPath, "cert.pem"), filepath.Join(certPath, "key.pem"))
	if err != nil {
		return nil, err
	}

	// Create docker client.
	client, err := docker.NewClient(h.String())
	if err != nil {
		return nil, err
	}

	// Specify our custom HTTP client with TLS-enabled transport.
	client.HTTPClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:      roots,
				Certificates: []tls.Certificate{cert},
			},
		},
	}
	return client, nil
}

// newDockerClient creates a new Docker client configured to reach Docker at the
// DOCKER_HOST env var, or the default /var/run/docker.sock socket if unset.
func newDockerClient() (*docker.Client, error) {
	dockerEndpoint := os.Getenv("DOCKER_HOST")
	if dockerEndpoint == "" {
		dockerEndpoint = "unix:///var/run/docker.sock"
	} else if !strings.HasPrefix(dockerEndpoint, "http") && !strings.HasPrefix(dockerEndpoint, "tcp") {
		dockerEndpoint = "http://" + dockerEndpoint
	} else if strings.HasPrefix(dockerEndpoint, "tcp") {
		certPath := os.Getenv("DOCKER_CERT_PATH")
		if certPath != "" {
			return newTLSDockerClient(certPath, dockerEndpoint)
		}
	}
	return docker.NewClient(dockerEndpoint)
}
