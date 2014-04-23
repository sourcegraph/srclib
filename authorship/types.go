package authorship

import (
	"time"

	"github.com/sourcegraph/go-nnz/nnz"
	"sourcegraph.com/sourcegraph/srcgraph/graph"
	"sourcegraph.com/sourcegraph/srcgraph/repo"
)

type AuthorshipInfo struct {
	AuthorEmail    string    `db:"author_email"`
	LastCommitDate time.Time `db:"last_commit_date"`

	// LastCommitID is the commit ID of the last commit that this author made to
	// the thing that this info describes.
	LastCommitID string `db:"last_commit_id"`
}

type SymbolAuthorship struct {
	AuthorshipInfo

	// Exported is whether the symbol is exported.
	Exported bool

	Chars           int     `db:"chars"`
	CharsProportion float64 `db:"chars_proportion"`
}

type SymbolAuthor struct {
	UID   nnz.Int
	Email nnz.String
	SymbolAuthorship
}

// RefAuthorship describes the authorship information (author email, date, and
// commit ID) of a ref. A ref may only have one author.
type RefAuthorship struct {
	graph.RefKey
	AuthorshipInfo
}

type SymbolClient struct {
	UID   nnz.Int
	Email nnz.String

	AuthorshipInfo

	// UseCount is the number of times this person referred to the symbol.
	UseCount int `db:"use_count"`
}

type RepositoryAuthorship struct {
	AuthorshipInfo

	// SymbolCount is the number of symbols that this author
	// contributed to this repository (where "contributed to" means "committed
	// any hunk of code to the definition of").
	SymbolCount int `db:"symbol_count"`

	SymbolsProportion float64 `db:"symbols_proportion"`

	// ExportedSymbolCount is the number of exported symbols that this author
	// contributed to this repository (where "contributed to" means "committed
	// any hunk of code to the definition of").
	ExportedSymbolCount int `db:"exported_symbol_count"`

	ExportedSymbolsProportion float64 `db:"exported_symbols_proportion"`

	// TODO(sqs): add "most recently contributed exported symbol"
}

type RepoContribution struct {
	RepoURI repo.URI `db:"repo"`
	RepositoryAuthorship
}

type RepositoryClientship struct {
	AuthorshipInfo

	// SymbolRepo is the repository that defines symbols that this author
	// referred to, in code committed to another repository.
	SymbolRepo repo.URI `db:"symbol_repo"`

	// RefCount is the number of references this author made in this repository to SymbolRepo.
	RefCount int `db:"ref_count"`
}

type RepoAuthor struct {
	UID   nnz.Int
	Email nnz.String
	RepositoryAuthorship
}

type RepoClient struct {
	UID   nnz.Int
	Email nnz.String
	RepositoryClientship
}

// RepoUsageByClient describes a repository whose code is referenced by a
// specific person.
type RepoUsageByClient struct {
	// SymbolRepo is the repository that defines the code that was referenced.
	// It's called SymbolRepo because "Repo" usually refers to the repository
	// whose analysis created this linkage (i.e., the repository that contains
	// the reference).
	SymbolRepo repo.URI `db:"symbol_repo" json:"-"`

	RefCount int `db:"ref_count"`

	AuthorshipInfo
}

// RepoUsageOfAuthor describes a repository referencing code committed by a
// specific person.
type RepoUsageOfAuthor struct {
	Repo repo.URI `json:"-"`

	RefCount int `db:"ref_count"`

	AuthorshipInfo
}
