package doc

import (
	"errors"
	"net/url"
	"path/filepath"
	"strings"

	"sourcegraph.com/sourcegraph/repo"
	"sourcegraph.com/sourcegraph/vcsfs"
)

type Service interface {
	GetFormattedReadme(repo *repo.Repository) (string, error)
}

// vcsfsService is an implementation of Service that uses the global vcsfs to
// fetch files.
type vcsfsService struct{}

var Default Service = &vcsfsService{}

var ErrNoReadme = errors.New("no readme found in repository")

// GetFormattedReadme returns repo's HTML-formatted readme, or an empty string
// and ErrNoReadme if the repository has no README.
func (_ *vcsfsService) GetFormattedReadme(repo *repo.Repository) (formattedReadme string, err error) {
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

type MockService struct {
	GetFormattedReadme_ func(repo *repo.Repository) (string, error)
}

var _ Service = MockService{}

func (s MockService) GetFormattedReadme(repo *repo.Repository) (string, error) {
	if s.GetFormattedReadme_ == nil {
		return "", nil
	}
	return s.GetFormattedReadme_(repo)
}

var readmeNames = []string{
	"README.md",
	"README.rst",
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
