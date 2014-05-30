package python

import (
	"encoding/json"
	"path/filepath"

	"github.com/kr/fs"
	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/container"
	"sourcegraph.com/sourcegraph/srcgraph/scan"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
)

func init() {
	scan.Register("python", scan.DockerScanner{defaultPythonEnv})
	unit.Register(DistPackageDisplayName, &DistPackage{})
}

func (p *pythonEnv) BuildScanner(dir string, c *config.Repository) (*container.Command, error) {
	dockerfile, err := p.pydepDockerfile()
	if err != nil {
		return nil, err
	}
	var cmd = []string{"pydep-run.py", "list", srcRoot}
	return &container.Command{
		Container: container.Container{
			Dockerfile: dockerfile,
			RunOptions: []string{"-v", dir + ":" + srcRoot},
			Cmd:        cmd,
		},
		Transform: func(orig []byte) ([]byte, error) {
			if len(orig) == 0 {
				return nil, nil
			}

			var pkgs []pkgInfo
			err := json.Unmarshal(orig, &pkgs)
			if err != nil {
				return nil, err
			}
			var units []*DistPackage
			for _, pkg := range pkgs {
				units = append(units, pkg.DistPackageWithFiles(pythonSourceFiles(pkg.RootDir)))
			}
			return json.Marshal(units)
		},
	}, nil
}

func (p *pythonEnv) UnmarshalSourceUnits(data []byte) ([]unit.SourceUnit, error) {
	if len(data) == 0 {
		return nil, nil
	}

	var distPackages []*DistPackage
	err := json.Unmarshal(data, &distPackages)
	if err != nil {
		return nil, err
	}

	units := make([]unit.SourceUnit, len(distPackages))
	for i, p := range distPackages {
		units[i] = p
	}

	return units, nil
}

func pythonSourceFiles(dir string) (files []string) {
	walker := fs.Walk(dir)
	for walker.Step() {
		if err := walker.Err(); err == nil && !walker.Stat().IsDir() && filepath.Ext(walker.Path()) == ".py" {
			file, _ := filepath.Rel(dir, walker.Path())
			files = append(files, file)
		}
	}
	return
}
