package ruby

import (
	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/container"
	"sourcegraph.com/sourcegraph/srcgraph/dep2"
	"sourcegraph.com/sourcegraph/srcgraph/task2"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
)

const rubyGemTargetType = "ruby-gem"

func init() {
	dep2.RegisterLister(&gem{}, dep2.DockerLister{defaultRubyEnv})
	dep2.RegisterLister(&app{}, dep2.DockerLister{defaultRubyEnv})
	dep2.RegisterResolver(rubyGemTargetType, defaultRubyEnv)
}

func (e *rubyEnv) BuildLister(dir string, unit unit.SourceUnit, c *config.Repository, x *task2.Context) (*container.Command, error) {

	return &container.Command{
		Container: container.Container{
			Dockerfile: e.DepDockerfile(),
			RunOptions: []string{"-v", dir + ":" + srcRoot},
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

func (e *rubyEnv) Resolve(dep *dep2.RawDependency, c *config.Repository, x *task2.Context) (*dep2.ResolvedTarget, error) {
	return nil, nil
}

func (e *rubyEnv) DepDockerfile() []byte {
	return nil
}
