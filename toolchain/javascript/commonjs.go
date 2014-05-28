package javascript

import "sourcegraph.com/sourcegraph/srcgraph/unit"

const commonJSPackageUnitType = "CommonJSPackage"

func init() {
	unit.Register(commonJSPackageUnitType, &CommonJSPackage{})
}

type CommonJSPackage struct {
	// If the field names of CommonJSPackage change, you need to EITHER (1)
	// update commonjs-findpkgs or (2) add a Transform func in the scanner to
	// map from the commonjs-findpkgs output to []*CommonJSPackage.

	// Dir is the directory that immediately contains the package.json
	// file (or would if one existed).
	Dir string

	// PackageJSONFile is the path to the package.json file, or empty if none
	// exists.
	PackageJSONFile string

	// Package is the parsed package.json file. We only read into a subset of
	// the fields.
	Package struct {
		Name string
	}

	LibFiles  []string
	TestFiles []string
}

func (p CommonJSPackage) Name() string    { return p.Package.Name }
func (p CommonJSPackage) RootDir() string { return p.Dir }
func (p CommonJSPackage) sourceFiles() []string {
	return append(append([]string{}, p.LibFiles...), p.TestFiles...)
}
func (p CommonJSPackage) Paths() []string {
	f := p.sourceFiles()
	if p.PackageJSONFile != "" {
		f = append(f, p.PackageJSONFile)
	}
	return f
}
