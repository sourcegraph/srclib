package repo

import (
	"database/sql/driver"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

// URI identifies a repository.
type URI string

// IsGitHubRepository returns true iff this repository is hosted on GitHub.
func (u URI) IsGitHubRepository() bool {
	return strings.HasPrefix(strings.ToLower(string(u)), "github.com/")
}

// IsGoogleCodeRepository returns true iff this repository is hosted on Google
// Code (code.google.com).
func (u URI) IsGoogleCodeRepository() bool {
	return strings.HasPrefix(strings.ToLower(string(u)), "code.google.com/p/")
}

// GitHubURL returns the https://github.com/USER/REPO URL for this repository.
func (u URI) GitHubURL() string {
	return "https://" + string(u)
}

// MakeURI converts a repository clone URL, such as
// "git://github.com/user/repo.git", to a normalized URI string, such as
// "github.com/user/repo".
func MakeURI(cloneURL string) URI {
	if cloneURL == "" {
		panic("MakeURI: empty clone URL")
	}

	url, err := url.Parse(cloneURL)
	if err != nil {
		panic(fmt.Sprintf("MakeURI(%q): %s", cloneURL, err))
	}

	path := strings.TrimSuffix(url.Path, ".git")
	path = filepath.Clean(path)
	path = strings.TrimSuffix(path, "/")
	return URI(strings.ToLower(url.Host) + path)
}

// URIEqual returns true if a and b are equal, based on a case insensitive
// comparison.
func URIEqual(a, b URI) bool {
	return strings.ToLower(string(a)) == strings.ToLower(string(b))
}

// Scan implements database/sql.Scanner.
func (u *URI) Scan(v interface{}) error {
	if v == nil {
		*u = ""
		return nil
	} else if data, ok := v.([]byte); ok {
		*u = URI(data)
		return nil
	}
	return fmt.Errorf("%T.Scan failed: %v", u, v)
}

// Value implements database/sql/driver.Valuer
func (u URI) Value() (driver.Value, error) {
	return string(u), nil
}

// URIs is a wrapper type for a slice of URIs.
type URIs []URI

// Strings returns the URIs as strings.
func (us URIs) Strings() []string {
	s := make([]string, len(us))
	for i, u := range us {
		s[i] = string(u)
	}
	return s
}
