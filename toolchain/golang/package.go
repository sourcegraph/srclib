package golang

import (
	"path/filepath"

	"sourcegraph.com/sourcegraph/srcgraph/unit"
)

const goPackageUnitType = "GoPackage"

func init() {
	unit.Register("GoPackage", &Package{})
}

type Package struct {
	Dir        string `toml:"dir"`
	ImportPath string `toml:"import_path"`
	Files      []string
}

func (p Package) Name() string    { return p.ImportPath }
func (p Package) RootDir() string { return p.Dir }
func (p Package) Paths() []string {
	paths := make([]string, len(p.Files))
	for i, f := range p.Files {
		paths[i] = filepath.Join(p.Dir, f)
	}
	return paths
}
