package unit

import (
	"encoding/json"
	"fmt"
	"reflect"

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

// SourceUnit decodes u's Data JSON field to the SourceUnit it represents, using
// the source unit registered as u.UnitType.
func (u *RepoSourceUnit) SourceUnit() (SourceUnit, error) {
	if u.UnitType == "" {
		return nil, fmt.Errorf(`source unit is missing "UnitType"`)
	}
	if emptyInstance, registered := Types[u.UnitType]; registered {
		typed := reflect.New(reflect.TypeOf(emptyInstance).Elem()).Interface()
		if err := json.Unmarshal(u.Data, typed); err != nil {
			return nil, err
		}
		return typed.(SourceUnit), nil
	}
	return nil, fmt.Errorf("unrecognized source unit type %q", u.UnitType)
}
