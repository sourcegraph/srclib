package plan

import (
	"path/filepath"

	"sourcegraph.com/sourcegraph/srclib/buildstore"
	"sourcegraph.com/sourcegraph/srclib/graph2"
	"sourcegraph.com/sourcegraph/srclib/unit"
)

func RepositoryCommitDataFilename(emptyData interface{}) string {
	return buildstore.DataTypeSuffix(emptyData)
}

func SourceUnitDataFilename(emptyData interface{}, u *unit.SourceUnit) string {
	return filepath.Join(u.Name, u.Type+"."+buildstore.DataTypeSuffix(emptyData))
}

func SourceUnitDataFilename2(emptyData interface{}, u *graph2.Unit) string {
	return filepath.Join(u.UnitName, u.UnitType+"."+buildstore.DataTypeSuffix(emptyData))
}
