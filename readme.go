package doc

import (
	"errors"
	"github.com/sqs/gorp"
	"net/url"
	"path/filepath"
	"sourcegraph.com/sourcegraph/repo"
	"sourcegraph.com/sourcegraph/vcsfs"
	"strings"
)

var ErrNoReadme = errors.New("no readme found in repository")

// GetFormattedReadme returns repo's HTML-formatted readme, or an empty string
// and ErrNoReadme if the repository has no README.
func GetFormattedReadme(dbh gorp.SqlExecutor, repo *repo.Repository) (formattedReadme string, err error) {
	cloneURL, err := url.Parse(repo.CloneURL)
	if err != nil {
		return "", err
	}
	src, path, err := vcsfs.GetFirstExistingFile(repo.VCS, cloneURL, repo.RevSpecOrDefault(), readmeNames)
	if err != nil {
		return "", ErrNoReadme
	}
	return ToHTML(readmeFormats[strings.ToLower(filepath.Ext(path))], string(src))
}

var readmeNames = []string{
	"README.md",
	"ReadMe.md",
	"Readme.md",
	"readme.md",
	"README.markdown",
	"ReadMe.markdown",
	"readme.markdown",
	"README",
	"ReadMe",
	"Readme",
	"readme",
	"README.rdoc",
	"README.txt",
	"ReadMe.txt",
	"readme.txt",
	"README.rst",
	"ReadMe.rst",
	"Readme.rst",
	"readme.rst",
}

var readmeFormats = map[string]Format{
	".md":       Markdown,
	".markdown": Markdown,
	".mdown":    Markdown,
	".rdoc":     Markdown, // TODO(sqs): actually implement RDoc
	".txt":      Text,
	".text":     Text,
	"":          Text,
	".ascii":    Text,
	".rst":      ReStructuredText,
}
