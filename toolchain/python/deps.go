package python

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"
	"github.com/beyang/cheerio"

	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/container"
	"sourcegraph.com/sourcegraph/srcgraph/dep2"
	"sourcegraph.com/sourcegraph/srcgraph/task2"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
)

func init() {
	dep2.RegisterLister(&pythonPackage{}, dep2.DockerLister{&pythonDependencyHandler{}})
	dep2.RegisterResolver(pythonRequirementTargetType, &pythonDependencyHandler{})
}

type pythonDependencyHandler struct{}

func (p *pythonDependencyHandler) BuildLister(dir string, unit unit.SourceUnit, c *config.Repository, x *task2.Context) (*container.Command, error) {
	dockerfile, err := p.dockerfile()
	if err != nil {
		return nil, err
	}

	containerSrcDir := "/src"
	return &container.Command{
		Container: container.Container{
			Dockerfile: dockerfile,
			RunOptions: []string{"-v", dir + ":" + containerSrcDir},
			Cmd:        []string{"cheerio", "reqsdir", containerSrcDir},
		},
		Transform: func(orig []byte) ([]byte, error) {
			var cheerioReqs []cheerioReq
			json.NewDecoder(bytes.NewReader(orig)).Decode(&cheerioReqs)

			deps := make([]*dep2.RawDependency, len(cheerioReqs))
			for i, chreq := range cheerioReqs {
				deps[i] = &dep2.RawDependency{
					TargetType: pythonRequirementTargetType,
					Target:     pythonRequirement{Name: chreq.Name, Version: chreq.Version, Constraint: versionConstraint(chreq.Constraint)},
				}
			}
			return json.Marshal(deps)
		},
	}, nil
}

func (p *pythonDependencyHandler) Resolve(dep *dep2.RawDependency, c *config.Repository, x *task2.Context) (*dep2.ResolvedTarget, error) {
	switch dep.TargetType {
	case pythonRequirementTargetType:
		pythonRequirement := dep.Target.(pythonRequirement)
		repoURL, err := cheerio.DefaultPyPI.FetchSourceRepoURL(pythonRequirement.Name)
		if err != nil {
			return nil, err
		}
		toUnit := &pythonPackage{name: pythonRequirement.Name}
		return &dep2.ResolvedTarget{
			ToRepoCloneURL: repoURL,
			ToUnit:         toUnit.Name(),
			ToUnitType:     unit.Type(toUnit),
		}, nil
	default:
		return nil, fmt.Errorf("Unexpected target type for Python %+v", dep.TargetType)
	}
}

type cheerioReq struct {
	Name       string
	Constraint string
	Version    string
}

func (l *pythonDependencyHandler) dockerfile() ([]byte, error) {
	// TODO: change once cheerio is ported to python
	var buf bytes.Buffer
	template.Must(template.New("").Parse(baseDockerfile)).Execute(&buf, struct {
		GoVersionString string
		GOPATH          string
	}{
		GoVersionString: "go1.2.1",
		GOPATH:          containerGOPATH,
	})
	return buf.Bytes(), nil
}

// pythonRequirement represents a Python dependency such as those declared in requirements.txt
type pythonRequirement struct {
	Name       string
	Version    string
	Constraint versionConstraint
}

type versionConstraint string

const (
	PyReq_LessThan         versionConstraint = "<"
	PyReq_LessThanEqual                      = "<="
	PyReq_NotEqual                           = "!="
	PyReq_Equal                              = "=="
	PyReq_GreaterThanEqual                   = ">="
	PyReq_GreaterThan                        = ">"
)

const pythonRequirementTargetType = "python-requirement"
const containerGOPATH = "/tmp/sg/gopath"
const baseDockerfile = `FROM ubuntu:13.10
RUN apt-get update
RUN apt-get install -qy curl
RUN apt-get install -qy git

# Install Go {{.GoVersionString}}.
RUN curl -o /tmp/golang.tgz https://go.googlecode.com/files/{{.GoVersionString}}.linux-amd64.tar.gz
RUN tar -xzf /tmp/golang.tgz -C /usr/local
ENV GOROOT /usr/local/go
ENV GOPATH {{.GOPATH}}

# Add "go" to the PATH.
ENV PATH {{.GOPATH}}/bin:/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

RUN go get github.com/beyang/cheerio/...
RUN go install github.com/beyang/cheerio/cmd/cheerio # TODO: versioning
`
