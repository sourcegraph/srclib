package client

import (
	"fmt"

	"github.com/sourcegraph/vcsstore/vcsclient"

	"sourcegraph.com/sourcegraph/api_router"
)

// RepositoryTreeService communicates with the Sourcegraph API endpoints that
// fetch file and directory entries in repositories.
type RepositoryTreeService interface {
	Get(entry TreeEntrySpec, opt *RepositoryTreeGetOptions) (*TreeEntry, Response, error)
}

type repositoryTreeService struct {
	client *Client
}

var _ RepositoryTreeService = &repositoryTreeService{}

type TreeEntrySpec struct {
	Repo RepositorySpec
	Path string
}

func (s *TreeEntrySpec) RouteVars() map[string]string {
	return map[string]string{"RepoURI": s.Repo.URI, "Rev": s.Repo.CommitID, "Path": s.Path}
}

func (s TreeEntrySpec) String() string {
	return fmt.Sprintf("%s: %s (rev %q)", s.Repo.URI, s.Path, s.Repo.CommitID)
}

// TreeEntry is a file or directory in a repository, with additional feedback
// from the formatting operation (if Formatted is true in the options).
type TreeEntry struct {
	*vcsclient.TreeEntry

	// FormatResult is only set if this TreeEntry is a file.
	FormatResult *FormatResult `json:",omitempty"`
}

// FormatResult contains information about and warnings from the formatting
// operation (if Formatted is true in the options).
type FormatResult struct {
	// TooManyRefs indicates that the file being formatted exceeded the maximum
	// number of refs that are linked. Only the first NumRefs refs are linked.
	TooManyRefs bool `json:",omitempty"`

	// NumRefs is the number of refs that were linked in this file. If the total
	// number of refs in the file exceeds the (server-defined) limit, NumRefs is
	// capped at the limit.
	NumRefs int
}

type RepositoryTreeGetOptions struct {
	Formatted bool
}

func (s *repositoryTreeService) Get(entry TreeEntrySpec, opt *RepositoryTreeGetOptions) (*TreeEntry, Response, error) {
	url, err := s.client.url(api_router.RepositoryTreeEntry, entry.RouteVars(), opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var entry_ *TreeEntry
	resp, err := s.client.Do(req, &entry_)
	if err != nil {
		return nil, resp, err
	}

	return entry_, resp, nil
}

type MockRepositoryTreeService struct {
	Get_ func(entry TreeEntrySpec, opt *RepositoryTreeGetOptions) (*TreeEntry, Response, error)
}

var _ RepositoryTreeService = MockRepositoryTreeService{}

func (s MockRepositoryTreeService) Get(entry TreeEntrySpec, opt *RepositoryTreeGetOptions) (*TreeEntry, Response, error) {
	if s.Get_ == nil {
		return nil, &HTTPResponse{}, nil
	}
	return s.Get_(entry, opt)
}
