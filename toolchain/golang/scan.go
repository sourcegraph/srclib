package golang

import (
	"bytes"
	"encoding/json"
	"path/filepath"

	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/container"
	"sourcegraph.com/sourcegraph/srcgraph/scan"
	"sourcegraph.com/sourcegraph/srcgraph/task2"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
)

func init() {
	scan.Register("go", scan.DockerScanner{defaultGoVersion})
}

func (v *goVersion) BuildScanner(dir string, c *config.Repository, x *task2.Context) (*container.Command, error) {
	goConfig := v.goConfig(c)

	dockerfile, err := v.baseDockerfile()
	if err != nil {
		return nil, err
	}

	containerDir := filepath.Join(containerGOPATH, "src", goConfig.BaseImportPath)
	cont := container.Container{
		Dockerfile: dockerfile,
		RunOptions: []string{"-v", dir + ":" + containerDir},
		Cmd:        []string{"go", "list", goConfig.BaseImportPath + "/..."},
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
			units := make([]unit.SourceUnit, len(lines))
			for i, line := range lines {
				importPath := string(line)
				dir, err := filepath.Rel(goConfig.BaseImportPath, importPath)
				if err != nil {
					return nil, err
				}
				units[i] = Package{
					Dir:        dir,
					ImportPath: importPath,
				}
			}
			return json.Marshal(units)
		},
	}
	return &cmd, nil
}

func (v *goVersion) UnmarshalSourceUnits(data []byte) ([]unit.SourceUnit, error) {
	if len(data) == 0 {
		return nil, nil
	}

	var goPackages []*Package
	err := json.Unmarshal(data, &goPackages)
	if err != nil {
		return nil, err
	}

	units := make([]unit.SourceUnit, len(goPackages))
	for i, p := range goPackages {
		units[i] = p
	}

	return units, nil
}
