package ruby

import (
	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/container"
	"sourcegraph.com/sourcegraph/srcgraph/grapher2"
	"sourcegraph.com/sourcegraph/srcgraph/task2"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
)

func init() {
	grapher2.Register(&gem{}, grapher2.DockerGrapher{defaultRubyEnv})
	grapher2.Register(&app{}, grapher2.DockerGrapher{defaultRubyEnv})
}

func (p *rubyEnv) BuildGrapher(dir string, unit unit.SourceUnit, c *config.Repository, x *task2.Context) (*container.Command, error) {
	return &container.Command{
		Container: container.Container{
			RunOptions: []string{"-v", dir + ":" + srcRoot},
			Dockerfile: p.grapherDockerfile(),
			Cmd:        []string{"echo", "TODO"}, // TODO
			Stderr:     x.Stderr,
			Stdout:     x.Stdout,
		},
		Transform: func(orig []byte) ([]byte, error) {
			// TODO
			return nil, nil
		},
	}, nil
}

func (p *rubyEnv) grapherDockerfile() []byte {
	return nil
}
