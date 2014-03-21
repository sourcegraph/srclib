//+build ignore

package javascript

import (
	"bytes"
	"encoding/json"
	"path/filepath"

	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/container"
	"sourcegraph.com/sourcegraph/srcgraph/scan"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
)

func init() {
	unit.Register("NPMPackage", NPMPackage{})
	scan.Register("npm", npmPackageScanner{})
}

type NPMPackage struct {
	PackageJSONFile string
	SourceFiles     []string
}

func (p NPMPackage) ID() string      { return "npm:" + p.PackageJSONFile }
func (p NPMPackage) Name() string    { return filepath.Dir(p.PackageJSONFile) }
func (p NPMPackage) Paths() []string { return p.SourceFiles }

type npmPackageScanner struct{}

func (s npmPackageScanner) Scan(dir string, c *config.Repository) (*container.Command, error) {
	buildfile := []byte(`FROM ubuntu:13.10
RUN apt-get update
RUN apt-get install -qy nodejs npm
`)

	containerDir := "/tmp/src"
	cont := container.Container{
		Dockerfile: buildfile,
		RunOptions: []string{"-v", dir + ":" + containerDir},
		Cmd:        []string{"find", containerDir, "-name", "package.json"},
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

func (s npmPackageScanner) UnmarshalSourceUnits(data []byte) ([]unit.SourceUnit, error) {
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
