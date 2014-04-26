package client

import (
	"fmt"
	"html/template"
	"net/url"

	"sourcegraph.com/sourcegraph/api_router"
	"sourcegraph.com/sourcegraph/srcgraph/authorship"
	"sourcegraph.com/sourcegraph/srcgraph/graph"
	"sourcegraph.com/sourcegraph/srcgraph/person"
	"sourcegraph.com/sourcegraph/srcgraph/repo"
)

type SymbolsService interface {
	Get(symbol SymbolSpec, opt *GetSymbolOptions) (*Symbol, *Response, error)
	List(opt *SymbolListOptions) ([]*Symbol, *Response, error)
	ListExamples(symbol SymbolSpec, opt *SymbolExampleListOptions) ([]*Example, *Response, error)
	ListAuthors(symbol SymbolSpec, opt *SymbolAuthorListOptions) ([]*AugmentedSymbolAuthor, *Response, error)
	ListClients(symbol SymbolSpec, opt *SymbolClientListOptions) ([]*AugmentedSymbolClient, *Response, error)
	ListDependentRepositories(symbol SymbolSpec, opt *SymbolDependentRepositoryListOptions) ([]*AugmentedRepoRef, *Response, error)
	ListImplementations(symbol SymbolSpec, opt *SymbolListImplementationsOptions) ([]*Symbol, *Response, error)
	ListInterfaces(symbol SymbolSpec, opt *SymbolListInterfacesOptions) ([]*Symbol, *Response, error)
	CountByRepository(repo RepositorySpec) (*graph.SymbolCounts, *Response, error)
}

type SymbolSpec struct {
	SID int64

	Repo     string
	UnitType string
	Unit     string
	Path     string
}

func (s *SymbolSpec) SymbolKey() graph.SymbolKey {
	return graph.SymbolKey{
		Repo:     repo.URI(s.Repo),
		UnitType: s.UnitType,
		Unit:     s.Unit,
		Path:     graph.SymbolPath(s.Path),
	}
}

func NewSymbolSpecFromSymbolKey(key graph.SymbolKey) SymbolSpec {
	return SymbolSpec{
		Repo:     string(key.Repo),
		UnitType: key.UnitType,
		Unit:     key.Unit,
		Path:     string(key.Path),
	}
}

type symbolsService struct {
	client *Client
}

var _ SymbolsService = &symbolsService{}

type Symbol struct {
	graph.Symbol

	Stat map[graph.StatType]int `json:",omitempty"`

	Doc      string           `json:",omitempty"`
	DefHTML  template.HTML    `json:",omitempty"`
	DocPages []*graph.DocPage `json:",omitempty"`
}

func (s *Symbol) SymbolSpec() SymbolSpec {
	spec := NewSymbolSpecFromSymbolKey(s.Symbol.SymbolKey)
	spec.SID = int64(s.Symbol.SID)
	return spec
}

func (s *Symbol) XRefs() int     { return s.Stat["xrefs"] }
func (s *Symbol) RRefs() int     { return s.Stat["rrefs"] }
func (s *Symbol) URefs() int     { return s.Stat["urefs"] }
func (s *Symbol) TotalRefs() int { return s.XRefs() + s.RRefs() + s.URefs() }

type GetSymbolOptions struct {
	Annotate bool `url:",omitempty"`
	DocPages bool `url:",omitempty"`
}

func (s *symbolsService) Get(symbol SymbolSpec, opt *GetSymbolOptions) (*Symbol, *Response, error) {
	var url *url.URL
	var err error
	if symbol.SID != 0 {
		url, err = s.client.url(api_router.SymbolBySID, map[string]string{"SID": fmt.Sprintf("%d", symbol.SID)}, opt)
	} else {
		url, err = s.client.url(api_router.Symbol, map[string]string{"RepoURI": symbol.Repo, "UnitType": symbol.UnitType, "Unit": symbol.Unit, "Path": symbol.Path}, opt)
	}
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var symbol_ *Symbol
	resp, err := s.client.Do(req, &symbol_)
	if err != nil {
		return nil, resp, err
	}

	return symbol_, resp, nil
}

type SymbolListOptions struct {
	RepositoryURI string `url:",omitempty"`
	Query         string `url:",omitempty"`

	Sort      string `url:",omitempty"`
	Direction string `url:",omitempty"`

	Kinds        []string `url:",omitempty,comma"`
	SpecificKind string   `url:",omitempty"`

	Scope     string `url:",omitempty"`
	Recursive bool   `url:",omitempty"`
	Exported  bool   `url:",omitempty"`
	Doc       bool   `url:",omitempty"`

	ListOptions
}

func (s *symbolsService) List(opt *SymbolListOptions) ([]*Symbol, *Response, error) {
	url, err := s.client.url(api_router.Symbols, nil, opt)
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

type Example struct {
	graph.Ref
	SrcHTML template.HTML
}

type Examples []*Example

func (r *Example) sortKey() string     { return fmt.Sprintf("%+v", r) }
func (vs Examples) Len() int           { return len(vs) }
func (vs Examples) Swap(i, j int)      { vs[i], vs[j] = vs[j], vs[i] }
func (vs Examples) Less(i, j int) bool { return vs[i].sortKey() < vs[j].sortKey() }

type SymbolExampleListOptions struct {
	Annotate bool

	ListOptions
}

func (s *symbolsService) ListExamples(symbol SymbolSpec, opt *SymbolExampleListOptions) ([]*Example, *Response, error) {
	url, err := s.client.url(api_router.SymbolExamples, map[string]string{"RepoURI": symbol.Repo, "UnitType": symbol.UnitType, "Unit": symbol.Unit, "Path": symbol.Path}, opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var examples []*Example
	resp, err := s.client.Do(req, &examples)
	if err != nil {
		return nil, resp, err
	}

	return examples, resp, nil
}

type AugmentedSymbolAuthor struct {
	User *person.User
	*authorship.SymbolAuthor
}

type SymbolAuthorListOptions struct {
	ListOptions
}

func (s *symbolsService) ListAuthors(symbol SymbolSpec, opt *SymbolAuthorListOptions) ([]*AugmentedSymbolAuthor, *Response, error) {
	url, err := s.client.url(api_router.SymbolAuthors, map[string]string{"RepoURI": symbol.Repo, "UnitType": symbol.UnitType, "Unit": symbol.Unit, "Path": symbol.Path}, opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var authors []*AugmentedSymbolAuthor
	resp, err := s.client.Do(req, &authors)
	if err != nil {
		return nil, resp, err
	}

	return authors, resp, nil
}

type AugmentedSymbolClient struct {
	User *person.User
	*authorship.SymbolClient
}

type SymbolClientListOptions struct {
	ListOptions
}

func (s *symbolsService) ListClients(symbol SymbolSpec, opt *SymbolClientListOptions) ([]*AugmentedSymbolClient, *Response, error) {
	url, err := s.client.url(api_router.SymbolClients, map[string]string{"RepoURI": symbol.Repo, "UnitType": symbol.UnitType, "Unit": symbol.Unit, "Path": symbol.Path}, opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var clients []*AugmentedSymbolClient
	resp, err := s.client.Do(req, &clients)
	if err != nil {
		return nil, resp, err
	}

	return clients, resp, nil
}

type AugmentedRepoRef struct {
	Repo  *repo.Repository
	Count int
}

type SymbolDependentRepositoryListOptions struct {
	ListOptions
}

func (s *symbolsService) ListDependentRepositories(symbol SymbolSpec, opt *SymbolDependentRepositoryListOptions) ([]*AugmentedRepoRef, *Response, error) {
	url, err := s.client.url(api_router.SymbolDependents, map[string]string{"RepoURI": symbol.Repo, "UnitType": symbol.UnitType, "Unit": symbol.Unit, "Path": symbol.Path}, opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var dependents []*AugmentedRepoRef
	resp, err := s.client.Do(req, &dependents)
	if err != nil {
		return nil, resp, err
	}

	return dependents, resp, nil
}

type SymbolListImplementationsOptions struct {
	ListOptions
}

func (s *symbolsService) ListImplementations(symbol SymbolSpec, opt *SymbolListImplementationsOptions) ([]*Symbol, *Response, error) {
	url, err := s.client.url(api_router.SymbolImplementations, map[string]string{"RepoURI": symbol.Repo, "UnitType": symbol.UnitType, "Unit": symbol.Unit, "Path": symbol.Path}, opt)
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

type SymbolListInterfacesOptions struct {
	ListOptions
}

func (s *symbolsService) ListInterfaces(symbol SymbolSpec, opt *SymbolListInterfacesOptions) ([]*Symbol, *Response, error) {
	url, err := s.client.url(api_router.SymbolInterfaces, map[string]string{"RepoURI": symbol.Repo, "UnitType": symbol.UnitType, "Unit": symbol.Unit, "Path": symbol.Path}, opt)
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

func (s *symbolsService) CountByRepository(repo RepositorySpec) (*graph.SymbolCounts, *Response, error) {
	url, err := s.client.url(api_router.RepositorySymbolCounts, map[string]string{"RepoURI": repo.URI}, nil)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var counts *graph.SymbolCounts
	resp, err := s.client.Do(req, &counts)
	if err != nil {
		return nil, resp, err
	}

	return counts, resp, nil
}

type MockSymbolsService struct {
	Get_                       func(symbol SymbolSpec, opt *GetSymbolOptions) (*Symbol, *Response, error)
	List_                      func(opt *SymbolListOptions) ([]*Symbol, *Response, error)
	ListExamples_              func(symbol SymbolSpec, opt *SymbolExampleListOptions) ([]*Example, *Response, error)
	ListAuthors_               func(symbol SymbolSpec, opt *SymbolAuthorListOptions) ([]*AugmentedSymbolAuthor, *Response, error)
	ListClients_               func(symbol SymbolSpec, opt *SymbolClientListOptions) ([]*AugmentedSymbolClient, *Response, error)
	ListDependentRepositories_ func(symbol SymbolSpec, opt *SymbolDependentRepositoryListOptions) ([]*AugmentedRepoRef, *Response, error)
	ListImplementations_       func(symbol SymbolSpec, opt *SymbolListImplementationsOptions) ([]*Symbol, *Response, error)
	ListInterfaces_            func(symbol SymbolSpec, opt *SymbolListInterfacesOptions) ([]*Symbol, *Response, error)
	CountByRepository_         func(repo RepositorySpec) (*graph.SymbolCounts, *Response, error)
}

var _ SymbolsService = MockSymbolsService{}

func (s MockSymbolsService) Get(symbol SymbolSpec, opt *GetSymbolOptions) (*Symbol, *Response, error) {
	if s.Get_ == nil {
		return nil, &Response{}, nil
	}
	return s.Get_(symbol, opt)
}

func (s MockSymbolsService) List(opt *SymbolListOptions) ([]*Symbol, *Response, error) {
	if s.List_ == nil {
		return nil, &Response{}, nil
	}
	return s.List_(opt)
}

func (s MockSymbolsService) ListExamples(symbol SymbolSpec, opt *SymbolExampleListOptions) ([]*Example, *Response, error) {
	if s.ListExamples_ == nil {
		return nil, &Response{}, nil
	}
	return s.ListExamples_(symbol, opt)
}

func (s MockSymbolsService) ListAuthors(symbol SymbolSpec, opt *SymbolAuthorListOptions) ([]*AugmentedSymbolAuthor, *Response, error) {
	if s.ListAuthors_ == nil {
		return nil, &Response{}, nil
	}
	return s.ListAuthors_(symbol, opt)
}

func (s MockSymbolsService) ListClients(symbol SymbolSpec, opt *SymbolClientListOptions) ([]*AugmentedSymbolClient, *Response, error) {
	if s.ListClients_ == nil {
		return nil, &Response{}, nil
	}
	return s.ListClients_(symbol, opt)
}

func (s MockSymbolsService) ListDependentRepositories(symbol SymbolSpec, opt *SymbolDependentRepositoryListOptions) ([]*AugmentedRepoRef, *Response, error) {
	if s.ListDependentRepositories_ == nil {
		return nil, &Response{}, nil
	}
	return s.ListDependentRepositories_(symbol, opt)
}

func (s MockSymbolsService) ListImplementations(symbol SymbolSpec, opt *SymbolListImplementationsOptions) ([]*Symbol, *Response, error) {
	if s.ListImplementations_ == nil {
		return nil, &Response{}, nil
	}
	return s.ListImplementations_(symbol, opt)
}

func (s MockSymbolsService) ListInterfaces(symbol SymbolSpec, opt *SymbolListInterfacesOptions) ([]*Symbol, *Response, error) {
	if s.ListInterfaces_ == nil {
		return nil, &Response{}, nil
	}
	return s.ListInterfaces_(symbol, opt)
}

func (s MockSymbolsService) CountByRepository(repo RepositorySpec) (*graph.SymbolCounts, *Response, error) {
	if s.CountByRepository_ == nil {
		return &graph.SymbolCounts{}, &Response{}, nil
	}
	return s.CountByRepository_(repo)
}
