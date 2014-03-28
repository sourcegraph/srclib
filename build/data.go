package build

import (
	"fmt"
	"path/filepath"
	"reflect"

	"sourcegraph.com/sourcegraph/srcgraph/buildstore"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
	"sourcegraph.com/sourcegraph/srcgraph/util2/makefile"
)

type RepositoryCommitDataFile struct {
	DataType reflect.Type
}

func (f *RepositoryCommitDataFile) Name() string { return buildstore.DataTypeSuffix(f.DataType) }

type SourceUnitDataFile struct {
	DataType reflect.Type
	Unit     unit.SourceUnit
}

func (f *SourceUnitDataFile) Name() string {
	return filepath.Clean(fmt.Sprintf("%s_%s", unit.MakeID(f.Unit), buildstore.DataTypeSuffix(f.DataType)))
}

// isDataFile returns true iff the makefile.File is one of the build data file
// types (RepositoryCommitDataFile, SourceUnitDataFile, etc.) and false
// otherwise (e.g., if it's just a normal file).
func isDataFile(f makefile.File) bool {
	switch f.(type) {
	case *RepositoryCommitDataFile:
		return true
	case *SourceUnitDataFile:
		return true
	}
	return false
}
