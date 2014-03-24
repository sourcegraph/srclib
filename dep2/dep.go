package dep2

import (
	"sourcegraph.com/sourcegraph/repo"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
)

type ResolvedDep struct {
	// FromRepo is the repository from which this dependency originates.
	FromRepo repo.URI `db:"from_repo" json:",omitempty"`

	// FromCommitID is the VCS commit in the repository that this dep was found
	// in.
	FromCommitID string `db:"from_commit" json:",omitempty"`

	// FromUnit is the source unit ID from which this dependency originates.
	FromUnitID unit.ID `db:"from_unit"`

	*ResolvedTarget
}
