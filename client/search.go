package client

import "sourcegraph.com/sourcegraph/api_router"

type SearchService interface {
	Search(opt *SearchOptions) ([]*Symbol, Response, error)
}

type searchService struct {
	client *Client
}

var _ SearchService = &searchService{}

type SearchOptions struct {
	Query    string
	Exported bool `url:",omitempty"`
	Instant  bool `url:",omitempty"`
	ListOptions
}

func (s *searchService) Search(opt *SearchOptions) ([]*Symbol, Response, error) {
	url, err := s.client.url(api_router.Search, nil, opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var symbols []*Symbol
	resp, err := s.client.Do(req, &symbols)
	if err != nil {
		return nil, resp, err
	}

	return symbols, resp, nil
}

type MockSearchService struct {
	Search_ func(opt *SearchOptions) ([]*Symbol, Response, error)
}

var _ SearchService = MockSearchService{}

func (s MockSearchService) Search(opt *SearchOptions) ([]*Symbol, Response, error) {
	if s.Search_ == nil {
		return nil, &HTTPResponse{}, nil
	}
	return s.Search_(opt)
}
