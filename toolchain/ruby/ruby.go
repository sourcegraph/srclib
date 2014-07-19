package ruby

import (
	"bytes"
	"text/template"

	"sync"

	"github.com/sourcegraph/srclib/config"
	"github.com/sourcegraph/srclib/dep2"
	"github.com/sourcegraph/srclib/toolchain"
)

func init() {
	toolchain.Register("ruby", DefaultRubyVersion)
	config.Register("ruby", &Config{})
}

type Ruby struct {
	Version        string
	StdlibCloneURL string

	// resolveCache maps gem name to resolved dep target.
	resolveCache   map[string]*dep2.ResolvedTarget
	resolveCacheMu sync.Mutex
}

var DefaultRubyVersion = &Ruby{
	Version:        "2.0.0-p481",
	StdlibCloneURL: "git://github.com/ruby/ruby.git",
}

func (v *Ruby) baseDockerfile() ([]byte, error) {
	var buf bytes.Buffer
	err := template.Must(template.New("").Parse(baseDockerfile)).Execute(&buf, struct {
		Ruby   *Ruby
		GOPATH string
	}{
		Ruby: v,
	})
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

const baseDockerfile = `FROM ubuntu:14.04
RUN apt-get update -qq
RUN apt-get install -qqy curl

# Install Ruby {{.Ruby.Version}}
RUN curl -L https://get.rvm.io | bash -s stable
ENV PATH /usr/local/rvm/bin:$PATH
RUN rvm requirements
RUN rvm install {{.Ruby.Version}}
RUN rvm {{.Ruby.Version}} do gem install bundler --no-ri --no-rdoc

# Lots of gemspecs run git to list files, so it's necessary.
RUN apt-get install -qqy git
`

type Config struct {
	// OmitStdlib is whether to NOT load the analyzed graph of the Ruby standard
	// library when analyzing this repository. It should be true when analyzing
	// the Ruby stdlib itself.
	OmitStdlib bool
}

func (v *Ruby) rubyConfig(c *config.Repository) *Config {
	rubyConfig, _ := c.Global["ruby"].(*Config)
	if rubyConfig == nil {
		rubyConfig = new(Config)
	}
	return rubyConfig
}
