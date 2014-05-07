package javascript

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"strings"

	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/container"
	"sourcegraph.com/sourcegraph/srcgraph/dep2"
	"sourcegraph.com/sourcegraph/srcgraph/grapher2"
	"sourcegraph.com/sourcegraph/srcgraph/scan"
	"sourcegraph.com/sourcegraph/srcgraph/task2"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
)

func init() {
	unit.Register("NPMPackage", &NPMPackage{})
	scan.Register("npm", scan.DockerScanner{defaultNPM})
	dep2.RegisterLister(&NPMPackage{}, defaultNPM)
	dep2.RegisterResolver(npmDependencyTargetType, dep2.DockerResolver{defaultNPM})
	grapher2.Register(&NPMPackage{}, grapher2.DockerGrapher{defaultNPM})
}

type NPMPackage struct {
	PackageJSONFile string
	SourceFiles     []string
}

func (p NPMPackage) Name() string    { return filepath.Dir(p.PackageJSONFile) }
func (p NPMPackage) RootDir() string { return filepath.Dir(p.PackageJSONFile) }
func (p NPMPackage) Paths() []string { return p.SourceFiles }

type npmVersion struct{}

var defaultNPM = &npmVersion{}

func (v *npmVersion) baseDockerfile() ([]byte, error) {
	return []byte(baseNPMDockerfile), nil
}

const baseNPMDockerfile = `FROM ubuntu:14.04
RUN apt-get update
RUN apt-get install -qy nodejs npm`

// containerDir returns the directory in the Docker container to use for the
// local directory dir.
func containerDir(dir string) string {
	return filepath.Join("/tmp/sg", filepath.Base(dir))
}

func (v *npmVersion) BuildScanner(dir string, c *config.Repository, x *task2.Context) (*container.Command, error) {
	dockerfile, err := v.baseDockerfile()
	if err != nil {
		return nil, err
	}

	containerDir := containerDir(dir)
	cont := container.Container{
		Dockerfile: dockerfile,
		RunOptions: []string{"-v", dir + ":" + containerDir},
		Cmd:        []string{"find", containerDir, "-name", "package.json"},
		Stderr:     x.Stderr,
		Stdout:     x.Stdout,
	}
	cmd := container.Command{
		Container: cont,
		Transform: func(orig []byte) ([]byte, error) {
			if len(orig) == 0 {
				return nil, nil
			}

			lines := bytes.Split(bytes.TrimSpace(orig), []byte("\n"))
			units := make([]*NPMPackage, len(lines))
			for i, line := range lines {
				packageJSONFile := string(line)
				packageJSONFile, err := filepath.Rel(containerDir, packageJSONFile)
				if err != nil {
					return nil, err
				}
				units[i] = &NPMPackage{
					PackageJSONFile: packageJSONFile,
				}
			}
			return json.Marshal(units)
		},
	}
	return &cmd, nil
}

func (v *npmVersion) UnmarshalSourceUnits(data []byte) ([]unit.SourceUnit, error) {
	if len(data) == 0 {
		return nil, nil
	}

	var npmPackages []*NPMPackage
	err := json.Unmarshal(data, &npmPackages)
	if err != nil {
		return nil, err
	}

	units := make([]unit.SourceUnit, len(npmPackages))
	for i, p := range npmPackages {
		units[i] = p
	}

	return units, nil
}

// npmDependency is a name/version pair that represents an NPM dependency. This
// pair corresponds to the object property/value pairs in package.json
// "dependency" objects.
type npmDependency struct {
	// Name is the package name of the dependency.
	Name string

	// Spec is the specifier of the version, which can be an NPM version number,
	// a tarball URL, a git/hg clone URL, etc.
	Spec string
}

const npmDependencyTargetType = "npm-dep"

func (v *npmVersion) BuildResolver(dep *dep2.RawDependency, c *config.Repository, x *task2.Context) (*container.Command, error) {
	var npmDep npmDependency
	j, _ := json.Marshal(dep.Target)
	json.Unmarshal(j, &npmDep)

	dockerfile, err := v.baseDockerfile()
	if err != nil {
		return nil, err
	}
	dockerfile = append(dockerfile, []byte("\n\nRUN npm install -g deptool@~0.0.2\n")...)

	cmd := container.Command{
		Container: container.Container{
			Dockerfile: dockerfile,
			Cmd:        []string{"nodejs", "/usr/local/bin/npm-deptool", npmDep.Name + "@" + npmDep.Spec},
			Stderr:     x.Stderr,
			Stdout:     x.Stdout,
		},
		Transform: func(orig []byte) ([]byte, error) {
			// resolvedDep is output from npm-deptool.
			type npmDeptoolOutput struct {
				ResolvedURL string `json:"_resolved"`
				ID          string `json:"_id"`
				Repository  struct {
					Type string
					URL  string
				}
			}
			var resolvedDeps map[string]npmDeptoolOutput
			err := json.Unmarshal(orig, &resolvedDeps)
			if err != nil {
				return nil, err
			}

			if len(resolvedDeps) == 0 {
				return nil, fmt.Errorf("npm-deptool did not output anything for raw dependency %+v", dep)
			}
			if len(resolvedDeps) != 1 {
				return nil, fmt.Errorf("npm-deptool unexpectedly returned %d deps for raw dependency %+v", len(resolvedDeps), dep)
			}

			var resolvedDep npmDeptoolOutput
			for _, v := range resolvedDeps {
				resolvedDep = v
			}

			var toRepoCloneURL, toRevSpec string
			if strings.HasPrefix(resolvedDep.ResolvedURL, "https://registry.npmjs.org/") {
				// known npm package, so the repository refers to it
				toRepoCloneURL = resolvedDep.Repository.URL
			} else {
				// external tarball, git repo url, etc., so the repository might
				// refer to the source repo (if this is a fork) or not be
				// present at all
				u, err := url.Parse(resolvedDep.ResolvedURL)
				if err != nil {
					return nil, err
				}
				toRevSpec = u.Fragment

				u.Fragment = ""
				toRepoCloneURL = u.String()
			}

			return json.Marshal(&dep2.ResolvedTarget{
				ToRepoCloneURL:  toRepoCloneURL,
				ToUnitType:      unit.Type((&NPMPackage{})),
				ToUnit:          ".",
				ToVersionString: resolvedDep.ID,
				ToRevSpec:       toRevSpec,
			})
		},
	}
	return &cmd, nil
}

// List reads the "dependencies" key in the NPM package's package.json file and
// outputs the properties as raw dependencies.
func (v *npmVersion) List(dir string, unit unit.SourceUnit, c *config.Repository, x *task2.Context) ([]*dep2.RawDependency, error) {
	pkg := unit.(*NPMPackage)
	pkgFile := filepath.Join(dir, pkg.PackageJSONFile)

	f, err := os.Open(pkgFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var pkgjson struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	err = json.NewDecoder(f).Decode(&pkgjson)
	if err != nil {
		return nil, err
	}

	rawDeps := make([]*dep2.RawDependency, len(pkgjson.Dependencies)+len(pkgjson.DevDependencies))
	i := 0
	addDeps := func(deps map[string]string) {
		for name, spec := range deps {
			rawDeps[i] = &dep2.RawDependency{
				FromFile:   pkg.PackageJSONFile,
				TargetType: npmDependencyTargetType,
				Target:     npmDependency{Name: name, Spec: spec},
			}
			i++
		}
	}
	addDeps(pkgjson.Dependencies)
	addDeps(pkgjson.DevDependencies)

	return rawDeps, nil
}

func (v *npmVersion) BuildGrapher(dir string, unit unit.SourceUnit, c *config.Repository, x *task2.Context) (*container.Command, error) {
	return &container.Command{}, nil
}
