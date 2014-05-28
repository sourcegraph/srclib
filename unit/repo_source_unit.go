package unit

import (
	"github.com/jmoiron/sqlx/types"

	"sourcegraph.com/sourcegraph/srcgraph/repo"
)

// A RepoSourceUnit is the "concrete" form of SourceUnit that includes
// information about which repository (and commit) the source unit exists in. In
// general, type SourceUnit is used during analysis of a single source unit and
// type RepoSourceUnit is used afterwards (either in cross-source-unit analysis,
// such as cross-reference resolution, or in after-the-fact DB/API queries).
type RepoSourceUnit struct {
	Repo     repo.URI
	CommitID string `db:"commit_id"`
	UnitType string `db:"unit_type"`
	Unit     string

	// Data is the JSON of the underlying SourceUnit.
	Data types.JsonText
}
