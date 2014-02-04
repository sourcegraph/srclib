package doc

import (
	"errors"
	"github.com/sqs/gorp"
	"net/url"
	"sourcegraph.com/sourcegraph/repo"
	"sourcegraph.com/sourcegraph/vcsfs"
)

var ErrNoReadme = errors.New("no readme found in repository")

// GetFormattedReadme returns repo's HTML-formatted readme, or an empty string
// and ErrNoReadme if the repository has no README.
func GetFormattedReadme(dbh gorp.SqlExecutor, repo *repo.Repository) (formattedReadme string, err error) {
	for _, rd := range readmeNames {
		cloneURL, err := url.Parse(repo.CloneURL)
		if err != nil {
			return "", err
		}
		src, err := vcsfs.GetFile(repo.VCS, cloneURL, repo.RevSpecOrDefault(), rd.name)
		if err == nil {
			return ToHTML(rd.fmt, string(src))
		}
	}
	return "", ErrNoReadme
}

var readmeNames = []struct {
	name string
	fmt  Format
}{
	{"README.md", Markdown},
	{"ReadMe.md", Markdown},
	{"Readme.md", Markdown},
	{"readme.md", Markdown},
	{"README.markdown", Markdown},
	{"ReadMe.markdown", Markdown},
	{"readme.markdown", Markdown},
	{"README", Text},
	{"ReadMe", Text},
	{"Readme", Text},
	{"readme", Text},
	{"README.rdoc", Text},
	{"README.txt", Text},
	{"ReadMe.txt", Text},
	{"readme.txt", Text},
	{"README.rst", ReStructuredText},
	{"ReadMe.rst", ReStructuredText},
	{"Readme.rst", ReStructuredText},
	{"readme.rst", ReStructuredText},
}
