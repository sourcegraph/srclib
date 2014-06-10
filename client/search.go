package client

import (
	"sourcegraph.com/sourcegraph/api_router"
	"sourcegraph.com/sourcegraph/srcgraph/person"
)

// SearchService communicates with the search-related endpoints in
// the Sourcegraph API.
type SearchService interface {
	// Search searches the full index.
	Search(opt *SearchOptions) (*SearchResults, Response, error)
}

type SearchResults struct {
	Symbols      []*Symbol
	People       []*person.User
	Repositories []*Repository
}

func (r *SearchResults) Empty() bool {
	return len(r.Symbols) == 0 && len(r.People) == 0 && len(r.Repositories) == 0
}

// searchService implements SearchService.
type searchService struct {
	client *Client
}

var _ SearchService = &searchService{}

type SearchOptions struct {
	Query string `url:"q" schema:"q"`

	Symbols      bool
	Repositories bool
	People       bool

	ListOptions
}

func (s *searchService) Search(opt *SearchOptions) (*SearchResults, Response, error) {
	url, err := s.client.url(api_router.Search, nil, opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var results *SearchResults
	resp, err := s.client.Do(req, &results)
	if err != nil {
		return nil, resp, err
	}

	return results, resp, nil
}

type MockSearchService struct {
	Search_ func(opt *SearchOptions) (*SearchResults, Response, error)
}

var _ SearchService = MockSearchService{}

func (s MockSearchService) Search(opt *SearchOptions) (*SearchResults, Response, error) {
	if s.Search_ == nil {
		return nil, &HTTPResponse{}, nil
	}
	return s.Search_(opt)
}
