package build

import (
	"fmt"
	"path/filepath"

	"sourcegraph.com/sourcegraph/srcgraph/buildstore"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
)

func RepositoryCommitDataFilename(emptyData interface{}) string {
	return buildstore.DataTypeSuffix(emptyData)
}

func SourceUnitDataFilename(emptyData interface{}, u unit.SourceUnit) string {
	return filepath.Clean(fmt.Sprintf("%s_%s", unit.MakeID(u), buildstore.DataTypeSuffix(emptyData)))
}
