package client

import (
	"fmt"

	"github.com/sourcegraph/vcsstore/vcsclient"

	"sourcegraph.com/sourcegraph/api_router"
)

type RepositoryTreeService interface {
	Get(entry TreeEntrySpec, opt *RepositoryTreeGetOptions) (*vcsclient.TreeEntry, Response, error)
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

type RepositoryTreeGetOptions struct {
	Annotated bool
}

func (s *repositoryTreeService) Get(entry TreeEntrySpec, opt *RepositoryTreeGetOptions) (*vcsclient.TreeEntry, Response, error) {
	url, err := s.client.url(api_router.RepositoryTreeEntry, entry.RouteVars(), opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var entry_ *vcsclient.TreeEntry
	resp, err := s.client.Do(req, &entry_)
	if err != nil {
		return nil, resp, err
	}

	return entry_, resp, nil
}

type MockRepositoryTreeService struct {
	Get_ func(entry TreeEntrySpec, opt *RepositoryTreeGetOptions) (*vcsclient.TreeEntry, Response, error)
}

var _ RepositoryTreeService = MockRepositoryTreeService{}

func (s MockRepositoryTreeService) Get(entry TreeEntrySpec, opt *RepositoryTreeGetOptions) (*vcsclient.TreeEntry, Response, error) {
	if s.Get_ == nil {
		return nil, &HTTPResponse{}, nil
	}
	return s.Get_(entry, opt)
}
