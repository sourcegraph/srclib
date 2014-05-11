package doc

import (
	"errors"
	"io/ioutil"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/sourcegraph/vcsstore/vcsclient"
	"sourcegraph.com/sourcegraph/srcgraph/repo"
)

type ReadmeFormatter interface {
	GetFormattedReadme(repo *repo.Repository) (string, error)
}

func NewReadmeFormatter(o vcsclient.RepositoryOpener) ReadmeFormatter {
	return &readmeFormatter{o}
}

type readmeFormatter struct {
	VCSClient vcsclient.RepositoryOpener
}

var ErrNoReadme = errors.New("no readme found in repository")

// GetFormattedReadme returns repo's HTML-formatted readme, or an empty string
// and ErrNoReadme if the repository has no README.
func (s *readmeFormatter) GetFormattedReadme(repo *repo.Repository) (formattedReadme string, err error) {
	cloneURL, err := url.Parse(repo.CloneURL)
	if err != nil {
		return "", err
	}

	rc := s.VCSClient.Repository(string(repo.VCS), cloneURL)

	commitID, err := rc.ResolveBranch(repo.DefaultBranch)
	if err != nil {
		return "", err
	}

	fs, err := rc.FileSystem(commitID)
	if err != nil {
		return "", err
	}

	entries, err := fs.ReadDir(".")
	if err != nil {
		return "", err
	}

	for _, e := range entries {
		if lowerName := strings.ToLower(e.Name()); strings.HasPrefix(lowerName, "readme") {
			// found the readme
			f, err := fs.Open(e.Name())
			if err != nil {
				return "", err
			}
			defer f.Close()

			data, err := ioutil.ReadAll(f)
			if err != nil {
				return "", err
			}

			return ToHTML(readmeFormats[filepath.Ext(lowerName)], string(data))
		}
	}
	return "", ErrNoReadme
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
