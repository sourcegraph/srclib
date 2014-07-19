package build

import (
	"fmt"
	"path/filepath"

	"github.com/sourcegraph/srclib/buildstore"
	"github.com/sourcegraph/srclib/unit"
)

func RepositoryCommitDataFilename(emptyData interface{}) string {
	return buildstore.DataTypeSuffix(emptyData)
}

func SourceUnitDataFilename(emptyData interface{}, u unit.SourceUnit) string {
	return filepath.Clean(fmt.Sprintf("%s_%s", unit.MakeID(u), buildstore.DataTypeSuffix(emptyData)))
}
