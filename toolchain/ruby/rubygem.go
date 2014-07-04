package ruby

import (
	"path/filepath"

	"strings"

	"sourcegraph.com/sourcegraph/srcgraph/repo"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
)

const rubygemUnitType = "rubygem"

func init() {
	unit.Register(rubygemUnitType, &RubyGem{})
}

type RubyGem struct {
	// GemName is the Gem::Specification "name" field. We assume that a gem name
	// is unique among all gems in any given repository.
	GemName string

	// Version is the Gem::Specification "version" field.
	Version string

	// Summary is the Gem::Specification "summary" field.
	Summary string

	// Description_ is the Gem::Specification "description" field.
	Description_ string `json:"Description"`

	// Homepage is the Gem::Specification "homepage" field.
	Homepage string

	// GemSpecFile is the path to the *.gemspec file for this gem, relative to
	// the repository root.
	GemSpecFile string

	// Files is the Gem::Specification "files" field.
	Files []string

	// RequirePaths is the Gem::Specification "require_paths" field.
	RequirePaths []string
}

func (p RubyGem) Name() string    { return p.GemName }
func (p RubyGem) RootDir() string { return filepath.Dir(p.GemSpecFile) }
func (p RubyGem) sourceFiles() []string {
	var files []string
	for _, f := range p.Files {
		if strings.HasSuffix(f, ".rb") {
			files = append(files, f)
		}
	}
	return files
}
func (p RubyGem) Paths() []string {
	paths := append([]string{p.GemSpecFile}, p.Files...)
	return paths
}

// NameInRepository implements unit.Info.
func (p RubyGem) NameInRepository(defining repo.URI) string { return p.Name() }

// GlobalName implements unit.Info.
func (p RubyGem) GlobalName() string { return p.Name() }

// Description implements unit.Info.
func (p RubyGem) Description() string { return p.Description_ }

// Type implements unit.Info.
func (p RubyGem) Type() string { return "RubyGem" }
