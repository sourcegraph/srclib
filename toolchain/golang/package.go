package golang

import (
	"path/filepath"

	"strings"

	"sourcegraph.com/sourcegraph/srcgraph/repo"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
)

const goPackageUnitType = "GoPackage"

func init() {
	unit.Register("GoPackage", &Package{})
}

// A Package is a Go package directory (which represents a package and its
// XTestPackage, if any).
type Package struct {
	Dir        string
	ImportPath string

	// Files
	Files []string

	// PackageName is the Go package name (the name that comes after "package "
	// at the top of the package's files).
	PackageName string

	// IsStdlib is whether this package is a Go stdlib package (i.e., it is in
	// $GOROOT).
	IsStdlib bool `json:",omitempty"`

	// Doc is the first sentence of the package documentation.
	Doc string `json:",omitempty"`
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

// NameInRepository implements unit.Info.
func (p Package) NameInRepository(defining repo.URI) string {
	if p.IsStdlib {
		return p.ImportPath
	}
	return strings.TrimPrefix(p.ImportPath, filepath.Join(string(defining), "..")+"/")
}

// GlobalName implements unit.Info.
func (p Package) GlobalName() string { return p.ImportPath }

// Description implements unit.Info.
func (p Package) Description() string { return p.Doc }

// Type implements unit.Info.
func (p Package) Type() string {
	if p.PackageName == "main" {
		return "Go command"
	}
	return "Go package"
}
