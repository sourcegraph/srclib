package dep2

import "sourcegraph.com/sourcegraph/repo"

type ResolvedDep struct {
	// FromRepo is the repository from which this dependency originates.
	FromRepo repo.URI `db:"from_repo" json:",omitempty"`

	// FromCommitID is the VCS commit in the repository that this dep was found
	// in.
	FromCommitID string `db:"from_commit_id" json:",omitempty"`

	// FromUnit is the source unit name from which this dependency originates.
	FromUnit string `db:"from_unit"`

	// FromUnitType is the source unit type from which this dependency originates.
	FromUnitType string `db:"from_unit_type"`

	// ToRepo is the repository containing the source unit that is depended on.
	ToRepo repo.URI `db:"to_repo"`

	// ToUnit is the name of the source unit that is depended on.
	ToUnit string `db:"to_unit"`

	// ToUnitType is the type of the source unit that is depended on.
	ToUnitType string `db:"to_unit_type"`

	// ToVersion is the version of the dependent repository (if known),
	// according to whatever version string specifier is used by FromRepo's
	// dependency management system.
	ToVersionString string `db:"to_version_string"`

	// ToRevSpec specifies the desired VCS revision of the dependent repository
	// (if known).
	ToRevSpec string `db:"to_rev_spec"`
}
