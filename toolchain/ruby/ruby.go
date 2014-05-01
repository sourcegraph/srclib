package ruby

import (
	"bytes"
	"text/template"

	"sourcegraph.com/sourcegraph/srcgraph/toolchain"
)

const srcRoot = "/src" // path to source in container
const rubyGemTargetType = "ruby-gem"

func init() {
	toolchain.Register("ruby", defaultRubyEnv)
}

type rubyEnv struct {
	RubyVersion string
	RDepVersion string
}

var defaultRubyEnv = &rubyEnv{
	RubyVersion: "2.1.1",
	RDepVersion: "0.0.5a",
}

// rdep datastructures
type metadata_t struct {
	Type         string         `json:"type"`
	Path         string         `json:"path,omitempty"`
	Name         string         `json:"name,omitempty"`
	Version      string         `json:"version,omitempty"`
	Dependencies []dependency_t `json:"dependencies,omitempty"`
}

type dependency_t struct {
	Name         string   `json:"name"`
	SourceURL    string   `json:"source_url,omitempty"`
	Requirements []string `json:"requirements,omitempty"`
}

// Note: git is needed because some projects (e.g., sinatra) call it from gemspec files
var rdepDockerfileTemplate = template.Must(template.New("").Parse(`FROM ubuntu:13.10
RUN apt-get update

RUN apt-get install -qy curl
RUN apt-get install -qy git

RUN apt-get install -qy ruby
RUN curl -sSL https://get.rvm.io | bash -s stable --ruby
RUN /bin/bash -l -c "rvm requirements"
RUN /bin/bash -l -c "rvm reload"
RUN /bin/bash -l -c "rvm install {{.RubyVersion}}"
RUN echo "\nrvm use {{.RubyVersion}} &> /dev/null" >> /.bash_profile

RUN gem install rdep -v {{.RDepVersion}}
`))

func (e *rubyEnv) rdepDockerfile() ([]byte, error) {
	var b bytes.Buffer
	err := rdepDockerfileTemplate.Execute(&b, e)
	return b.Bytes(), err
}
