package authorship

import (
	"encoding/json"
	"time"

	"github.com/sourcegraph/go-nnz/nnz"
	"sourcegraph.com/sourcegraph/srclib/graph"
	"sourcegraph.com/sourcegraph/srclib/repo"
)

type AuthorshipInfo struct {
	AuthorEmail    string    `db:"author_email"`
	LastCommitDate time.Time `db:"last_commit_date"`

	// LastCommitID is the commit ID of the last commit that this author made to
	// the thing that this info describes.
	LastCommitID string `db:"last_commit_id"`
}

type DefAuthorship struct {
	AuthorshipInfo

	// Exported is whether the def is exported.
	Exported bool

	Chars           int     `db:"chars"`
	CharsProportion float64 `db:"chars_proportion"`
}

type DefAuthor struct {
	UID   nnz.Int
	Email nnz.String
	DefAuthorship
}

// RefAuthorship describes the authorship information (author email, date, and
// commit ID) of a ref. A ref may only have one author.
type RefAuthorship struct {
	graph.RefKey
	AuthorshipInfo
}

func (a *RefAuthorship) sortKey() string {
	// PERF TODO(sqs): slow
	b, err := json.Marshal(a)
	if err != nil {
		panic(err.Error())
	}
	return string(b)
}

type DefClient struct {
	UID   nnz.Int
	Email nnz.String

	AuthorshipInfo

	// UseCount is the number of times this person referred to the def.
	UseCount int `db:"use_count"`
}

type AuthorStats struct {
	AuthorshipInfo

	// DefCount is the number of defs that this author contributed (where
	// "contributed" means "committed any hunk of code to source code files").
	DefCount int `db:"def_count"`

	DefsProportion float64 `db:"defs_proportion"`

	// ExportedDefCount is the number of exported defs that this author
	// contributed (where "contributed to" means "committed any hunk of code to
	// source code files").
	ExportedDefCount int `db:"exported_def_count"`

	ExportedDefsProportion float64 `db:"exported_defs_proportion"`

	// TODO(sqs): add "most recently contributed exported def"
}

func (a *AuthorStats) sortKey() string {
	// PERF TODO(sqs): slow
	b, err := json.Marshal(a)
	if err != nil {
		panic(err.Error())
	}
	return string(b)
}

type RepoContribution struct {
	RepoURI repo.URI `db:"repo"`
	AuthorStats
}

type ClientStats struct {
	AuthorshipInfo

	// DefRepo is the repository that defines defs that this client
	// referred to.
	DefRepo repo.URI `db:"def_repo"`

	// DefUnitType and DefUnit are the unit in DefRepo that defines
	// defs that this client referred to. If DefUnitType == "" and
	// DefUnit == "", then this ClientStats is an aggregate of this client's
	// refs to all units in DefRepo.
	DefUnitType nnz.String `db:"def_unit_type"`
	DefUnit     nnz.String `db:"def_unit"`

	// RefCount is the number of references this client made in this repository
	// to DefRepo.
	RefCount int `db:"ref_count"`
}

func (a *ClientStats) sortKey() string {
	// PERF TODO(sqs): slow
	b, err := json.Marshal(a)
	if err != nil {
		panic(err.Error())
	}
	return string(b)
}

type RepoAuthor struct {
	UID   nnz.Int
	Email nnz.String
	AuthorStats
}

type RepoClient struct {
	UID   nnz.Int
	Email nnz.String
	ClientStats
}

// RepoUsageByClient describes a repository whose code is referenced by a
// specific person.
type RepoUsageByClient struct {
	// DefRepo is the repository that defines the code that was referenced.
	// It's called DefRepo because "Repo" usually refers to the repository
	// whose analysis created this linkage (i.e., the repository that contains
	// the reference).
	DefRepo repo.URI `db:"def_repo"`

	RefCount int `db:"ref_count"`

	AuthorshipInfo
}

// RepoUsageOfAuthor describes a repository referencing code committed by a
// specific person.
type RepoUsageOfAuthor struct {
	Repo repo.URI

	RefCount int `db:"ref_count"`
}
