package ruby

import (
	"bytes"
	"encoding/json"
	"log"
	"text/template"

	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/container"
	"sourcegraph.com/sourcegraph/srcgraph/graph"
	"sourcegraph.com/sourcegraph/srcgraph/grapher2"
	"sourcegraph.com/sourcegraph/srcgraph/repo"
	"sourcegraph.com/sourcegraph/srcgraph/task2"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
)

const RubyMRICloneURL = "git://github.com/ruby/ruby"

func init() {
	grapher2.Register(&gem{}, grapher2.DockerGrapher{defaultRubyEnv})
	grapher2.Register(&app{}, grapher2.DockerGrapher{defaultRubyEnv})
}

func (p *rubyEnv) BuildGrapher(dir string, u unit.SourceUnit, c *config.Repository, x *task2.Context) (*container.Command, error) {
	var buf bytes.Buffer
	err := yardCmdTemplate.Execute(&buf, struct {
		// don't use the RubyCore yardoc file for the Ruby core and stdlib (non-ext/)
		// files because then it think it has already processed the file and
		// would do nothing. But we should use it for Ruby ext/ so we don't emit
		// duplicate defns of core types (e.g., Kernel and Object).
		IsStandardLib bool
		SrcDir        string
		*rubyEnv
	}{
		IsStandardLib: (c.URI == repo.MakeURI(RubyMRICloneURL) && u.Name() == "."),
		SrcDir:        srcRoot,
		rubyEnv:       p,
	})
	if err != nil {
		return nil, err
	}
	cmd := []string{"/bin/bash", "-l", "-c", buf.String()}

	dockerfile, err := p.grapherDockerfile()
	if err != nil {
		return nil, err
	}

	return &container.Command{
		Container: container.Container{
			RunOptions: []string{"-v", dir + ":" + srcRoot},
			Dockerfile: dockerfile,
			Cmd:        cmd,
			Stderr:     x.Stderr,
			Stdout:     x.Stdout,
		},
		Transform: func(orig []byte) ([]byte, error) {
			var o struct {
				Objects    []*rubyObject `json:"objects"`
				References []*rubyRef    `json:"references"`
			}
			err := json.Unmarshal(orig, &o)
			if err != nil {
				return nil, err
			}

			o2 := grapher2.Output{
				Symbols: make([]*graph.Symbol, 0),
				Refs:    make([]*graph.Ref, 0),
				Docs:    make([]*graph.Doc, 0),
			}

			// TODO: apply additional transformations, handle std-lib test case
			seenSyms := make(map[graph.SymbolPath]*graph.Symbol)
			for _, rubyObj := range o.Objects {
				sym := rubyObj.toSymbol()
				if prevSym, seen := seenSyms[sym.Path]; seen {
					log.Printf("Skipping already seen symbol %+v -- other def is %+v", prevSym, sym)
					continue
				}
				seenSyms[sym.Path] = sym
				o2.Symbols = append(o2.Symbols, sym)

				if rubyObj.Docstring != "" {
					o2.Docs = append(o2.Docs, &graph.Doc{
						SymbolKey: sym.SymbolKey,
						Data:      rubyObj.Docstring,
					})
				}
			}

			for _, rubyRef := range o.References {
				ref, depGemName := rubyRef.toRef()
				if depGemName == StdlibGemNameSentinel {
					// TODO
				} else {
					// TODO
				}
				o2.Refs = append(o2.Refs, ref)
			}

			return json.Marshal(o2)
		},
	}, nil
}

func (p *rubyEnv) grapherDockerfile() ([]byte, error) {
	var buf bytes.Buffer
	err := yardDockerfileTemplate.Execute(&buf, p)
	return buf.Bytes(), err
}

var yardCmdTemplate = template.Must(template.New("").Parse(`
# install gems
cd {{.SrcDir}};
# TODO: make rdep handle all install cases (if Gemfile, then bundle, otherwise install via *.gemspec)
bundle install 1>&2;

export RUBY_FILES="$(ls {{.SrcDir}}/lib/**/*.rb) $(ls {{.SrcDir}}/test/**/*.rb)";  # TODO: get src and test directories from rdep/.gemspec?
# TODO(maybe?):
# - run YARD on all gems << is this an optimization or needed for correctness?... was YARD being run in the dep phase before?
# - pass gem yardoc files to this call of YARD
# - pass all ruby files to this call of YARD
/yard/bin/yard condense $RUBY_FILES;
# /yard/bin/yard condense {{if not .IsStandardLib}} -c TODO:include_stdlib_yardoc_db_path {{end}}
`))

var yardDockerfileTemplate = template.Must(template.New("").Parse(`FROM ubuntu:13.10
RUN apt-get update

RUN apt-get install -qy curl
RUN apt-get install -qy git

RUN apt-get install -qy ruby
RUN curl -sSL https://get.rvm.io | bash -s stable --ruby

RUN /bin/bash -l -c "rvm requirements"
RUN /bin/bash -l -c "rvm reload"
RUN /bin/bash -l -c "rvm install {{.RubyVersion}}"
RUN echo "\nrvm use {{.RubyVersion}} &> /dev/null" >> /.bash_profile
RUN /bin/bash -l -c "gem install bundler"

# Install Sourcegraph's fork of YARD
RUN git clone https://github.com/sourcegraph/yard /yard --single-branch --depth 1 --branch 0.0.1
WORKDIR /yard
RUN /bin/bash -l -c "bundle install"
WORKDIR /

ENV HOME /

# TODO(env): generate stdlib_yardoc_db and stdlib_yardoc_db_path
`))
