package person

import (
	"crypto/md5"
	"database/sql/driver"
	"fmt"
	"io"

	"github.com/sourcegraph/go-nnz/nnz"
	"sourcegraph.com/sourcegraph/srcgraph/db_common"
)

// User represents a user.
type User struct {
	// UID is the numeric primary key for a user.
	UID UID `db:"uid"`

	// GitHubID is the numeric ID of the GitHub user account corresponding to
	// this user.
	GitHubID nnz.Int `db:"github_id"`

	// Login is the user's username, which typically corresponds to the user's
	// GitHub login.
	Login string

	// Name is the (possibly empty) full name of the user.
	Name string

	// AvatarURL is the URL to an avatar image specified by the user.
	AvatarURL string

	// Location is the user's physical location (from their GitHub profile).
	Location string `json:",omitempty"`

	// Company is the user's company (from their GitHub profile).
	Company string `json:",omitempty"`

	// HomepageURL is the user's homepage or blog URL (from their GitHub
	// profile).
	HomepageURL string `db:"homepage_url" json:",omitempty"`

	// Transient is if this user was constructed on the fly and is not persisted
	// or resolved to a Sourcegraph/GitHub/etc. user.
	Transient bool `db:"-" json:",omitempty"`

	// UserProfileDisabled is whether the user profile should not be displayed
	// on the Web app.
	UserProfileDisabled bool `db:"user_profile_disabled" json:",omitempty"`

	// RegisteredAt is the date that the user registered. If the user has not
	// registered (i.e., we have processed their repos but they haven't signed
	// into Sourcegraph), it is null.
	RegisteredAt db_common.NullTime `db:"registered_at"`

	// GitHubOAuth2AccessToken is the user's GitHub access token.
	GitHubOAuth2AccessToken string `db:"github_oauth2_access_token" json:"-"` // don't write the OAuth2 access token to JSON

	OwnedReposCount       int `db:"owned_repos_count"`
	ContributedReposCount int `db:"contributed_repos_count"`
	AuthorsCount          int `db:"authors_count"`
	ClientsCount          int `db:"clients_count"`
	DependentsCount       int `db:"dependents_count"`
	DependenciesCount     int `db:"dependencies_count"`
}

// GitHubLogin returns the user's Login. They are the same for now, but callers
// that intend to get the GitHub login should call GitHubLogin() so that we can
// decouple the logins in the future if needed.
func (u *User) GitHubLogin() string {
	return u.Login
}

func (u *User) AvatarURLOfSize(width int) string {
	return u.AvatarURL + fmt.Sprintf("&s=%d", width)
}

// UID is the numeric primary key for a user.
type UID int

// Scan implements database/sql.Scanner.
func (x *UID) Scan(v interface{}) error {
	if data, ok := v.(int64); ok {
		*x = UID(data)
		return nil
	}
	return fmt.Errorf("%T.Scan failed: %v", x, v)
}

// Value implements database/sql/driver.Valuer.
func (x UID) Value() (driver.Value, error) {
	return int64(x), nil
}

// DefaultAvatarSize is the size, in pixels, of avatar images if no size is
// specified.
const DefaultAvatarSize = 128

// GravatarURL returns the URL to the Gravatar avatar image for email. If size
// is 0, DefaultAvatarSize is used.
func GravatarURL(email string, size uint16) string {
	if size == 0 {
		size = DefaultAvatarSize
	}
	h := md5.New()
	io.WriteString(h, email)
	return fmt.Sprintf("https://secure.gravatar.com/avatar/%x?s=%d&d=mm", h.Sum(nil), size)
}

// UserEmail is a row in the user_email DB table.
type UserEmail struct {
	UID   UID
	Email string
}
