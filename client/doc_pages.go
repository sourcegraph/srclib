package client

import (
	"sourcegraph.com/sourcegraph/api_router"
	"sourcegraph.com/sourcegraph/srcgraph/graph"
)

type DocPagesService interface {
	Get(docPage DocPageSpec, opt *GetDocPageOptions) (*graph.DocPage, Response, error)
}

type DocPageSpec struct {
	Repo RepositorySpec
	Path string

	// TODO(new-arch): what is the primary key for a doc page? update it here
	// when we figure out the best way to set a primary key for doc pages.
}

type docPagesService struct {
	client *Client
}

var _ DocPagesService = &docPagesService{}

type GetDocPageOptions struct{}

func (s *docPagesService) Get(docPage DocPageSpec, opt *GetDocPageOptions) (*graph.DocPage, Response, error) {
	url, err := s.client.url(api_router.RepositoryDocPage, map[string]string{"RepoURI": docPage.Repo.URI, "Path": docPage.Path}, opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var docPage_ *graph.DocPage
	resp, err := s.client.Do(req, &docPage_)
	if err != nil {
		return nil, resp, err
	}

	return docPage_, resp, nil
}

type MockDocPagesService struct {
	Get_ func(docPage DocPageSpec, opt *GetDocPageOptions) (*graph.DocPage, Response, error)
}

var _ DocPagesService = MockDocPagesService{}

func (s MockDocPagesService) Get(docPage DocPageSpec, opt *GetDocPageOptions) (*graph.DocPage, Response, error) {
	if s.Get_ == nil {
		return nil, &HTTPResponse{}, nil
	}
	return s.Get_(docPage, opt)
}
