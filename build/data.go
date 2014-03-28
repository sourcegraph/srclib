package build

import (
	"fmt"
	"path/filepath"
	"reflect"

	"sourcegraph.com/sourcegraph/srcgraph/buildstore"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
)

func RepositoryCommitDataFilename(dataType reflect.Type) string {
	return buildstore.DataTypeSuffix(dataType)
}

func SourceUnitDataFilename(dataType reflect.Type, u unit.SourceUnit) string {
	return filepath.Clean(fmt.Sprintf("%s_%s", unit.MakeID(u), buildstore.DataTypeSuffix(dataType)))
}
