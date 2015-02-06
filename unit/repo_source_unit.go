package unit

import "encoding/json"

// A RepoSourceUnit is the "concrete" form of SourceUnit that includes
// information about which repository (and commit) the source unit exists in. In
// general, type SourceUnit is used during analysis of a single source unit and
// type RepoSourceUnit is used afterwards (either in cross-source-unit analysis,
// such as cross-reference resolution, or in after-the-fact DB/API queries).
type RepoSourceUnit struct {
	Repo     string `json:",omitempty"`
	CommitID string `json:",omitempty"`
	UnitType string `json:",omitempty"`
	Unit     string `json:",omitempty"`

	// Data is the JSON of the underlying SourceUnit.
	Data json.RawMessage
}

// NewRepoSourceUnit creates an equivalent RepoSourceUnit from a
// SourceUnit.
//
// It does not set the returned source unit's Private field (because
// it can't tell if it is private from the underlying source unit
// alone).
//
// It also doesn't set CommitID (for the same reason).
func NewRepoSourceUnit(u *SourceUnit) (*RepoSourceUnit, error) {
	unitJSON, err := json.Marshal(u)
	if err != nil {
		return nil, err
	}
	return &RepoSourceUnit{
		Repo:     u.Repo,
		UnitType: u.Type,
		Unit:     u.Name,
		Data:     unitJSON,
	}, nil
}

// SourceUnit decodes u's Data JSON field to the SourceUnit it
// represents.
func (u *RepoSourceUnit) SourceUnit() (*SourceUnit, error) {
	var u2 *SourceUnit
	if err := json.Unmarshal(u.Data, &u2); err != nil {
		return nil, err
	}
	return u2, nil
}
