package ruby

import (
	"encoding/json"
	"path/filepath"

	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/container"
	"sourcegraph.com/sourcegraph/srcgraph/scan"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
)

func init() {
	scan.Register("ruby", scan.DockerScanner{DefaultRubyVersion})
}

func (v *Ruby) BuildScanner(dir string, c *config.Repository) (*container.Command, error) {
	dockerfile, err := v.baseDockerfile()
	if err != nil {
		return nil, err
	}

	containerDir := "/tmp/rubygem"
	cont := container.Container{
		Dockerfile:       dockerfile,
		AddDirs:          [][2]string{{dir, containerDir}},
		PreCmdDockerfile: []byte("\nRUN rvm all do gem install rubygems-find --version 0.0.4 --no-rdoc --no-ri\n"),
		Dir:              containerDir,
		Cmd:              []string{"rvm", "all", "do", "rubygems-find.rb"},
	}
	cmd := container.Command{
		Container: cont,
		Transform: func(orig []byte) ([]byte, error) {
			gems, err := gemspecJSONMapToRubyGems(orig, containerDir)
			if err != nil {
				return nil, err
			}
			return json.Marshal(gems)
		},
	}
	return &cmd, nil
}

func (v *Ruby) UnmarshalSourceUnits(data []byte) ([]unit.SourceUnit, error) {
	if len(data) == 0 {
		return nil, nil
	}

	var rubygems []*RubyGem
	err := json.Unmarshal(data, &rubygems)
	if err != nil {
		return nil, err
	}

	units := make([]unit.SourceUnit, len(rubygems))
	for i, p := range rubygems {
		units[i] = p
	}

	return units, nil
}

// gemspecJSONMapToRubyGems parses the JSON returned by rubygems-find.
func gemspecJSONMapToRubyGems(data []byte, containerDir string) ([]*RubyGem, error) {
	var gemsBySpec map[string]json.RawMessage
	err := json.Unmarshal(data, &gemsBySpec)
	if err != nil {
		return nil, err
	}

	var gems []*RubyGem
	for gemspecFile, gemSpecJSON := range gemsBySpec {
		gem, err := gemspecJSONToRubyGem(gemspecFile, gemSpecJSON, containerDir)
		if err != nil {
			return nil, err
		}
		gems = append(gems, gem)
	}

	return gems, nil
}

// gemspecJSONToRubyGem parses a single gemspec JSON returned by rubygems-find.
func gemspecJSONToRubyGem(gemspecFile string, gemspecJSON []byte, containerDir string) (*RubyGem, error) {
	// Make file paths relative to the repository root (i.e., containerDir).
	gemDir := filepath.Dir(gemspecFile)
	var err error
	gemDir, err = filepath.Rel(containerDir, gemDir)
	if err != nil {
		return nil, err
	}

	gemspecFile, err = filepath.Rel(containerDir, gemspecFile)
	if err != nil {
		return nil, err
	}

	var gemSpec struct {
		Name         string
		Version      string
		Homepage     string
		Summary      string
		Description  string
		Files        []string
		RequirePaths []string `json:"require_paths"`
	}
	err = json.Unmarshal(gemspecJSON, &gemSpec)
	if err != nil {
		return nil, err
	}
	return &RubyGem{
		GemName:      gemSpec.Name,
		Version:      gemSpec.Version,
		Summary:      gemSpec.Summary,
		Description_: gemSpec.Description,
		Homepage:     gemSpec.Homepage,
		GemSpecFile:  filepath.Clean(gemspecFile),
		Files:        addPrefixToPaths(gemDir, gemSpec.Files),
		RequirePaths: addPrefixToPaths(gemDir, gemSpec.RequirePaths),
	}, nil
}

func addPrefixToPaths(prefix string, paths []string) []string {
	for i, p := range paths {
		paths[i] = filepath.Join(prefix, p)
	}
	return paths
}
