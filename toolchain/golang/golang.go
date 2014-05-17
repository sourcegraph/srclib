package golang

import (
	"bytes"
	"text/template"

	"sync"

	"github.com/sourcegraph/go-vcsurl"
	"sourcegraph.com/sourcegraph/srcgraph/dep2"
	"sourcegraph.com/sourcegraph/srcgraph/toolchain"
)

func init() {
	// Register the Go toolchain.
	toolchain.Register("golang", defaultGoVersion)
}

// goVersion represents a Go release: where to download it, how to create graph
// references to it, etc.
type goVersion struct {
	// VersionString is the version string for this Go version, as listed at
	// https://code.google.com/p/go/downloads/list. (E.g., "go1.2.1" or
	// "go1.2rc5".)
	VersionString string

	RepositoryCloneURL string
	RepositoryVCS      vcsurl.VCS
	VCSRevision        string
	SourceUnitPrefix   string

	resolveCache   map[string]*dep2.ResolvedTarget
	resolveCacheMu sync.Mutex
}

var goVersions = map[string]*goVersion{
	"1.2.1": &goVersion{
		VersionString:      "go1.2.1",
		RepositoryCloneURL: "https://code.google.com/p/go",
		RepositoryVCS:      vcsurl.Mercurial,
		VCSRevision:        "go1.2.1",
		SourceUnitPrefix:   "src/pkg",
	},
}

var defaultGoVersion = goVersions["1.2.1"]

func (v *goVersion) baseDockerfile() ([]byte, error) {
	var buf bytes.Buffer
	err := template.Must(template.New("").Parse(baseDockerfile)).Execute(&buf, struct {
		GoVersion *goVersion
		GOPATH    string
	}{
		GoVersion: v,
		GOPATH:    containerGOPATH,
	})
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

const containerGOPATH = "/tmp/sg/gopath"

const baseDockerfile = `FROM ubuntu:14.04
RUN apt-get update -qq
RUN apt-get install -qqy curl

# Install Go {{.GoVersion.VersionString}}.
RUN curl -o /tmp/golang.tgz https://go.googlecode.com/files/{{.GoVersion.VersionString}}.linux-amd64.tar.gz
RUN tar -xzf /tmp/golang.tgz -C /usr/local
ENV GOROOT /usr/local/go

# Add "go" to the PATH.
ENV PATH /usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

ENV GOPATH {{.GOPATH}}
`

type baseBuild struct {
	Stdlib *goVersion
	GOPATH string
}
