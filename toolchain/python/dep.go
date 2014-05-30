package python

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"

	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/container"
	"sourcegraph.com/sourcegraph/srcgraph/dep2"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
)

func init() {
	dep2.RegisterLister(&DistPackage{}, dep2.DockerLister{defaultPythonEnv})
	dep2.RegisterResolver(pythonRequirementTargetType, defaultPythonEnv)
}

func (p *pythonEnv) BuildLister(dir string, u unit.SourceUnit, c *config.Repository) (*container.Command, error) {
	var dockerfile []byte
	var cmd []string
	var err error
	if c.URI == stdLibRepo {
		dockerfile = []byte(`FROM ubuntu:14.04`)
		cmd = []string{"echo", "[]"}
	} else {
		dockerfile, err = p.pydepDockerfile()
		if err != nil {
			return nil, err
		}
		cmd = []string{"pydep-run.py", "dep", filepath.Join(srcRoot, u.RootDir())}
	}

	return &container.Command{
		Container: container.Container{
			Dockerfile: dockerfile,
			RunOptions: []string{"-v", dir + ":" + srcRoot},
			Cmd:        cmd,
		},
		Transform: func(orig []byte) ([]byte, error) {
			var reqs []requirement
			err := json.NewDecoder(bytes.NewReader(orig)).Decode(&reqs)
			if err != nil {
				return nil, err
			}

			deps := make([]*dep2.RawDependency, 0)
			for _, req := range reqs {
				if req.RepoURL != "" { // cannot resolve dependencies with no clone URL
					deps = append(deps, &dep2.RawDependency{
						TargetType: pythonRequirementTargetType,
						Target:     req,
					})
				} else {
					log.Printf("(warn) ignoring dependency %+v because repo URL absent", req)
				}
			}
			return json.Marshal(deps)
		},
	}, nil
}

func (p *pythonEnv) Resolve(dep *dep2.RawDependency, c *config.Repository) (*dep2.ResolvedTarget, error) {
	switch dep.TargetType {
	case pythonRequirementTargetType:
		var req requirement
		reqJson, _ := json.Marshal(dep.Target)
		json.Unmarshal(reqJson, &req)

		toUnit := req.DistPackage()
		return &dep2.ResolvedTarget{
			ToRepoCloneURL: req.RepoURL,
			ToUnit:         toUnit.Name(),
			ToUnitType:     unit.Type(toUnit),
		}, nil
	default:
		return nil, fmt.Errorf("Unexpected target type for Python: %+v", dep.TargetType)
	}
}

const pythonRequirementTargetType = "python-requirement"
