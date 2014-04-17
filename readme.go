package doc

import (
	"errors"
	"net/url"
	"path/filepath"
	"strings"

	"sourcegraph.com/sourcegraph/repo"
	"sourcegraph.com/sourcegraph/vcsfs"
)

type ReadmeFormatter interface {
	GetFormattedReadme(repo *repo.Repository) (string, error)
}

type OnlineReadmeFormatter struct {
	VCSFS vcsfs.Service
}

var ErrNoReadme = errors.New("no readme found in repository")

// GetFormattedReadme returns repo's HTML-formatted readme, or an empty string
// and ErrNoReadme if the repository has no README.
func (s *OnlineReadmeFormatter) GetFormattedReadme(repo *repo.Repository) (formattedReadme string, err error) {
	cloneURL, err := url.Parse(repo.CloneURL)
	if err != nil {
		return "", err
	}
	src, path, err := s.VCSFS.GetFirstExistingFile(string(repo.VCS), cloneURL, repo.DefaultBranch, readmeNames)
	if err != nil {
		return "", ErrNoReadme
	}
	return ToHTML(readmeFormats[strings.ToLower(filepath.Ext(path))], string(src))
}

type MockReadmeFormatter struct {
	GetFormattedReadme_ func(repo *repo.Repository) (string, error)
}

var _ ReadmeFormatter = MockReadmeFormatter{}

func (s MockReadmeFormatter) GetFormattedReadme(repo *repo.Repository) (string, error) {
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
