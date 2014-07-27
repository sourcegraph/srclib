package plan

import (
	"fmt"
	"path/filepath"

	"github.com/sourcegraph/srclib/buildstore"
	"github.com/sourcegraph/srclib/unit"
)

func RepositoryCommitDataFilename(emptyData interface{}) string {
	return buildstore.DataTypeSuffix(emptyData)
}

func SourceUnitDataFilename(emptyData interface{}, u *unit.SourceUnit) string {
	return filepath.Clean(fmt.Sprintf("%s/%s.%s", u.Name, u.Type, buildstore.DataTypeSuffix(emptyData)))
}
