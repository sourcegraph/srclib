package repo

import (
	"database/sql/driver"
	"fmt"

	"github.com/sourcegraph/go-nnz/nnz"
	"sourcegraph.com/sourcegraph/srcgraph/person"
)

// RID is the numeric primary key for a repository.
type RID int

type Repository struct {
	// RID is the numeric primary key for a repository.
	RID RID

	// URI is a normalized identifier for this repository based on its primary
	// clone URL. E.g., "github.com/user/repo".
	URI URI

	// Name is the base name (the final path component) of the repository,
	// typically the name of the directory that the repository would be cloned
	// into. (For example, for git://example.com/foo.git, the name is "foo".)
	Name string

	// OwnerUserID is the account that owns this repository.
	OwnerUserID person.UID `db:"owner_user_id"`

	// OwnerGitHubUserID is the GitHub user ID of this repository's owner, if this
	// is a GitHub repository.
	OwnerGitHubUserID nnz.Int `db:"owner_github_user_id" json:",omitempty"`

	// Description is a brief description of the repository.
	Description string `json:",omitempty"`

	// VCS is the short name of the VCS system that this repository uses: "git"
	// or "hg".
	VCS VCS `db:"vcs"`

	// CloneURL is the URL used to clone the repository from its original host.
	CloneURL string `db:"clone_url"`

	// HomepageURL is the URL to the repository's homepage, if any.
	HomepageURL nnz.String `db:"homepage_url"`

	// DefaultBranch is the default VCS branch used (typically "master" for git
	// repositories and "default" for hg repositories).
	DefaultBranch string `db:"default_branch"`

	// Language is the primary programming language used in this repository.
	Language string 

	// GitHubStars is the number of stargazers this repository has on GitHub (or
	// 0 if it is not a GitHub repository).
	GitHubStars int `db:"github_stars"`

	// GitHubID is the GitHub ID of this repository. If a GitHub repository is
	// renamed, the ID remains the same and should be used to resolve across the
	// name change.
	GitHubID nnz.Int `db:"github_id" json:",omitempty"`

	// Disabled is whether this repo should not be downloaded and processed by the worker.
	Disabled bool `json:",omitempty"`

	// Deprecated repositories are labeled as such and hidden from global search results.
	Deprecated bool

	// Fork is whether this repository is a fork.
	Fork bool

	// Mirror is whether this repository is a mirror.
	Mirror bool
}

// IsGitHubRepository returns true iff this repository is hosted on GitHub.
func (r *Repository) IsGitHubRepository() bool {
	return r.URI.IsGitHubRepository()
}

type VCS string

const (
	Git VCS = "git"
	Hg  VCS = "hg"
)

// Scan implements database/sql.Scanner.
func (x *VCS) Scan(v interface{}) error {
	if data, ok := v.([]byte); ok {
		*x = VCS(data)
		return nil
	}
	return fmt.Errorf("%T.Scan failed: %v", x, v)
}

// Value implements database/sql/driver.Valuer
func (x VCS) Value() (driver.Value, error) {
	return string(x), nil
}

func MapByURI(repos []*Repository) map[URI]*Repository {
	repoMap := make(map[URI]*Repository, len(repos))
	for _, repo := range repos {
		repoMap[URI(repo.URI)] = repo
	}
	return repoMap
}

type Repositories []*Repository

func (rs Repositories) URIs() (uris []URI) {
	uris = make([]URI, len(rs))
	for i, r := range rs {
		uris[i] = r.URI
	}
	return
}
