package python

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"text/template"

	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/container"
	"sourcegraph.com/sourcegraph/srcgraph/dep2"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
)

func init() {
	dep2.RegisterLister(&FauxPackage{}, dep2.DockerLister{defaultPythonEnv})
	dep2.RegisterResolver(pythonRequirementTargetType, defaultPythonEnv)
}

func (p *pythonEnv) BuildLister(dir string, unit unit.SourceUnit, c *config.Repository) (*container.Command, error) {
	var dockerfile []byte
	var cmd []string
	var err error
	var runOpts = []string{"-v", dir + ":" + srcRoot}
	if c.URI == stdLibRepo {
		dockerfile = []byte(`FROM ubuntu:14.04`)
		cmd = []string{"echo", "[]"}
	} else {
		dockerfile, err = p.depDockerfile()
		if err != nil {
			return nil, err
		}
		cmd = []string{"pydep-run.py", "dep", srcRoot}
	}

	return &container.Command{
		Container: container.Container{
			Dockerfile: dockerfile,
			RunOptions: runOpts,
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

type requirement struct {
	ProjectName string      `json:"project_name"`
	UnsafeName  string      `json:"unsafe_name"`
	Key         string      `json:"key"`
	Specs       [][2]string `json:"specs"`
	Extras      []string    `json:"extras"`
	RepoURL     string      `json:"repo_url"`
	Packages    []string    `json:"packages"`
	Modules     []string    `json:"modules"`
	Resolved    bool        `json:"resolved"`
	Type        string      `json:"type"`
}

func (p *pythonEnv) Resolve(dep *dep2.RawDependency, c *config.Repository) (*dep2.ResolvedTarget, error) {
	switch dep.TargetType {
	case pythonRequirementTargetType:
		var req requirement
		reqJson, _ := json.Marshal(dep.Target)
		json.Unmarshal(reqJson, &req)

		toUnit := &FauxPackage{}
		return &dep2.ResolvedTarget{
			ToRepoCloneURL: req.RepoURL,
			ToUnit:         toUnit.Name(),
			ToUnitType:     unit.Type(toUnit),
		}, nil
	default:
		return nil, fmt.Errorf("Unexpected target type for Python: %+v", dep.TargetType)
	}
}

func (l *pythonEnv) depDockerfile() ([]byte, error) {
	var buf bytes.Buffer
	template.Must(template.New("").Parse(depDockerfile)).Execute(&buf, l)
	return buf.Bytes(), nil
}

const pythonRequirementTargetType = "python-requirement"
const depDockerfile = `FROM ubuntu:14.04
RUN apt-get update
RUN apt-get install -qy curl
RUN apt-get install -qy git
RUN apt-get install -qy {{.PythonVersion}}
RUN ln -s $(which {{.PythonVersion}}) /usr/bin/python
RUN curl https://raw.githubusercontent.com/pypa/pip/1.5.5/contrib/get-pip.py | python

RUN pip install git+git://github.com/sourcegraph/pydep.git@{{.PydepVersion}}
`
