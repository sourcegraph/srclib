package ruby

import (
	"encoding/json"
	"fmt"

	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/container"
	"sourcegraph.com/sourcegraph/srcgraph/scan"
	"sourcegraph.com/sourcegraph/srcgraph/task2"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
)

func init() {
	scan.Register("ruby", scan.DockerScanner{defaultRubyEnv})
	unit.Register("RubyGem", &gem{})
	unit.Register("RubyApp", &app{})
}

type gem struct {
	Dir     string
	GemName string
	Version string
}

func (u *gem) Name() string {
	return u.GemName
}

func (u *gem) RootDir() string {
	return u.Dir
}

func (u *gem) Paths() []string {
	return nil // TODO
}

type app struct {
	Dir string
}

func (u *app) Name() string {
	return ""
}

func (u *app) RootDir() string {
	return u.Dir
}

func (u *app) Paths() []string {
	return nil // TODO
}

func (e *rubyEnv) BuildScanner(dir string, c *config.Repository, x *task2.Context) (*container.Command, error) {
	scanDockerfile, err := e.rdepDockerfile()
	if err != nil {
		return nil, err
	}
	return &container.Command{
		Container: container.Container{
			Dockerfile: scanDockerfile,
			RunOptions: []string{"-v", dir + ":" + srcRoot},
			Cmd:        []string{"rdep", "--no-dep", "--scan", srcRoot},
			Stderr:     x.Stderr,
			Stdout:     x.Stdout,
		},
		Transform: func(orig []byte) ([]byte, error) {
			var metadata []metadata_t
			err := json.Unmarshal(orig, &metadata)
			if err != nil {
				return nil, err
			}

			var units []unit.SourceUnit
			for _, m := range metadata {
				switch m.Type {
				case "gem":
					units = append(units, &gem{Dir: m.Path, GemName: m.Name, Version: m.Version})
				case "app":
					units = append(units, &app{Dir: m.Path})
				default:
					return nil, fmt.Errorf("Unrecognized ruby project type %s", m.Type)
				}
			}
			return json.Marshal(units)
		},
	}, nil
}

func (e *rubyEnv) UnmarshalSourceUnits(data []byte) ([]unit.SourceUnit, error) {
	var unitMaps []map[string]string
	err := json.Unmarshal(data, &unitMaps)
	if err != nil {
		return nil, err
	}

	units := make([]unit.SourceUnit, len(unitMaps))
	for i, unitMap := range unitMaps {
		if unitMap["GemName"] != "" {
			units[i] = &gem{Dir: unitMap["Dir"], GemName: unitMap["GemName"], Version: unitMap["Version"]}
		} else {
			units[i] = &app{Dir: unitMap["Dir"]}
		}
	}
	return units, nil
}
