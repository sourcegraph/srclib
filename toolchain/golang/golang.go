package golang

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sync"
	"text/template"

	"github.com/sourcegraph/go-vcsurl"
	"sourcegraph.com/sourcegraph/srcgraph/buildstore"
	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/container"
	"sourcegraph.com/sourcegraph/srcgraph/dep2"
	"sourcegraph.com/sourcegraph/srcgraph/repo"
	"sourcegraph.com/sourcegraph/srcgraph/toolchain"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
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
	RepositoryURI      repo.URI
	RepositoryVCS      vcsurl.VCS
	VCSRevision        string
	BaseImportPath     string
	BasePkgDir         string

	resolveCache   map[string]*dep2.ResolvedTarget
	resolveCacheMu sync.Mutex
}

var goVersions = map[string]*goVersion{
	"1.3": &goVersion{
		VersionString:      "go1.3",
		RepositoryCloneURL: "https://code.google.com/p/go",
		RepositoryURI:      "code.google.com/p/go",
		RepositoryVCS:      vcsurl.Mercurial,
		VCSRevision:        "go1.3",
		BaseImportPath:     "code.google.com/p/go/src/pkg",
		BasePkgDir:         "src/pkg",
	},
}

var defaultGoVersion = goVersions["1.3"]

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

func (v *goVersion) containerForRepo(dir string, unit unit.SourceUnit, c *config.Repository) (*container.Container, error) {
	dockerfile, err := v.baseDockerfile()
	if err != nil {
		return nil, err
	}

	goConfig := v.goConfig(c)
	mountDir := filepath.Join(containerGOPATH, "src", goConfig.BaseImportPath)
	containerDir := mountDir

	var preCmdDockerfile []byte
	var addDirs, addFiles [][2]string
	if c.URI == v.RepositoryURI {
		// Go stdlib. This is fairly hacky. We want stdlib package paths to not
		// be prefixed with "code.google.com/p/go" everywhere (just
		dockerfile = append(dockerfile, []byte(fmt.Sprintf(`
# Adjust for Go stdlib
ENV GOROOT /tmp/go
RUN apt-get update -qq && apt-get install -qq build-essential mercurial
`))...)

		// Add all dirs needed for make.bash. Exclude dirs that change when
		// we build, so that we can take advantage of ADD caching and not
		// recompile the Go stdlib for each package.
		entries, err := ioutil.ReadDir(dir)
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if n := e.Name(); n == "." || n == "test" || n == "api" || n == ".." || n == "pkg" || n == "bin" || n == buildstore.BuildDataDirName {
				continue
			}
			if !e.Mode().IsDir() {
				continue
			}
			addDirs = append(addDirs, [2]string{e.Name(), filepath.Join("/tmp/go", e.Name())})
		}

		// We need to actually build the version of Go we want to analyze.
		preCmdDockerfile = []byte(fmt.Sprintf(`
RUN cd /tmp/go/src && ./make.bash
`))

		containerDir = "/tmp/go"
	}

	return &container.Container{
		Dockerfile:       dockerfile,
		RunOptions:       []string{"-v", dir + ":" + mountDir},
		PreCmdDockerfile: preCmdDockerfile,
		Dir:              containerDir,
		AddDirs:          addDirs,
		AddFiles:         addFiles,
	}, nil
}

const containerGOPATH = "/tmp/sg/gopath"

const baseDockerfile = `FROM ubuntu:14.04
RUN apt-get update -qq && apt-get install -qq curl

# Install Go {{.GoVersion.VersionString}}.
RUN curl -Lo /tmp/golang.tgz http://golang.org/dl/{{.GoVersion.VersionString}}.linux-amd64.tar.gz
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
