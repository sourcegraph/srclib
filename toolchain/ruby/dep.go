package ruby

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/container"
	"sourcegraph.com/sourcegraph/srcgraph/dep2"
	"sourcegraph.com/sourcegraph/srcgraph/task2"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
)

func init() {
	dep2.RegisterLister(&gem{}, dep2.DockerLister{defaultRubyEnv})
	dep2.RegisterLister(&app{}, dep2.DockerLister{defaultRubyEnv})
	dep2.RegisterResolver(rubyGemTargetType, defaultRubyEnv)
}

// TODO: make sure this handles the ruby standard lib and the emitted source unit is consistent with what's in grapher.go (special case)

func (e *rubyEnv) BuildLister(dir string, unit unit.SourceUnit, c *config.Repository, x *task2.Context) (*container.Command, error) {
	dockerfile, err := e.rdepDockerfile()
	if err != nil {
		return nil, err
	}
	return &container.Command{
		Container: container.Container{
			Dockerfile: dockerfile,
			RunOptions: []string{"-v", dir + ":" + srcRoot},
			Cmd:        []string{"rdep", filepath.Join(srcRoot, unit.RootDir())},
			Stderr:     x.Stderr,
			Stdout:     x.Stdout,
		},
		Transform: func(orig []byte) ([]byte, error) {
			var metadata metadata_t
			err := json.Unmarshal(orig, &metadata)
			if err != nil {
				return nil, err
			}

			deps := make([]*dep2.RawDependency, 0)
			for _, dependency := range metadata.Dependencies {
				if dependency.SourceURL != "" {
					deps = append(deps, &dep2.RawDependency{
						TargetType: rubyGemTargetType,
						Target:     dependency,
					})
				}
			}
			return json.Marshal(deps)
		},
	}, nil
}

func (e *rubyEnv) Resolve(dep *dep2.RawDependency, c *config.Repository, x *task2.Context) (*dep2.ResolvedTarget, error) {
	t, _ := json.Marshal(dep.Target)
	var target dependency_t
	json.Unmarshal(t, &target)

	switch dep.TargetType {
	case rubyGemTargetType:
		return &dep2.ResolvedTarget{
			ToRepoCloneURL: target.SourceURL,
			ToUnit:         target.Name,
			ToUnitType:     unit.Type(&gem{}),
		}, nil
	default:
		return nil, fmt.Errorf("Unexpected target type for Ruby: %+v", dep.TargetType)
	}
}
