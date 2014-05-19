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

// SymbolsService communicates with the symbol- and graph-related endpoints in
// the Sourcegraph API.
type SymbolsService interface {
	// Get fetches a symbol.
	Get(symbol SymbolSpec, opt *SymbolGetOptions) (*Symbol, Response, error)

	// List symbols.
	List(opt *SymbolListOptions) ([]*Symbol, Response, error)

	// Search symbols.
	Search(opt *SymbolSearchOptions) ([]*Symbol, Response, error)

	// Tree returns a tree of symbols.
	Tree(opt *SymbolTreeOptions) ([]*SymbolNode, Response, error)

	// ListExamples lists examples for symbol.
	ListExamples(symbol SymbolSpec, opt *SymbolListExamplesOptions) ([]*Example, Response, error)

	// ListExamples lists people who committed parts of symbol's definition.
	ListAuthors(symbol SymbolSpec, opt *SymbolListAuthorsOptions) ([]*AugmentedSymbolAuthor, Response, error)

	// ListClients lists people who use symbol in their code.
	ListClients(symbol SymbolSpec, opt *SymbolListClientsOptions) ([]*AugmentedSymbolClient, Response, error)

	// ListDependents lists repositories that use symbol in their code.
	ListDependents(symbol SymbolSpec, opt *SymbolListDependentsOptions) ([]*AugmentedSymbolDependent, Response, error)

	// ListImplementations lists types that implement symbol (an interface), according to
	// language-specific semantics.
	ListImplementations(symbol SymbolSpec, opt *SymbolListImplementationsOptions) ([]*Symbol, Response, error)

	// ListInterfaces lists interfaces that are implemented by symbol (a type),
	// according to language-specific semantics.
	ListInterfaces(symbol SymbolSpec, opt *SymbolListInterfacesOptions) ([]*Symbol, Response, error)

	// CountByRepository counts the symbols in repo grouped by kind.
	CountByRepository(repo RepositorySpec) (*graph.SymbolCounts, Response, error)
}

// SymbolSpec specifies a symbol. If SID == 0, then Repo, UnitType, and Unit
// must all be non-empty. (It is valid for Path to be empty.)
type SymbolSpec struct {
	SID int64

	Repo     string
	CommitID string
	UnitType string
	Unit     string
	Path     string
}

func (s *SymbolSpec) RouteVars() map[string]string {
	m := map[string]string{"RepoURI": s.Repo, "UnitType": s.UnitType, "Unit": s.Unit, "Path": s.Path}
	if s.CommitID != "" {
		m["Rev"] = s.CommitID
	}
	return m
}

// SymbolKey returns the symbol key specified by s, using the Repo, UnitType,
// Unit, and Path fields of s. If only s.SID is set, SymbolKey will panic.
func (s *SymbolSpec) SymbolKey() graph.SymbolKey {
	if s.Repo == "" {
		panic("Repo is empty")
	}
	if s.UnitType == "" {
		panic("UnitType is empty")
	}
	if s.Unit == "" {
		panic("Unit is empty")
	}
	return graph.SymbolKey{
		Repo:     repo.URI(s.Repo),
		CommitID: s.CommitID,
		UnitType: s.UnitType,
		Unit:     s.Unit,
		Path:     graph.SymbolPath(s.Path),
	}
}

// NewSymbolSpecFromSymbolKey returns a SymbolSpec that specifies the same
// symbol as the given key.
func NewSymbolSpecFromSymbolKey(key graph.SymbolKey) SymbolSpec {
	return SymbolSpec{
		Repo:     string(key.Repo),
		CommitID: key.CommitID,
		UnitType: key.UnitType,
		Unit:     key.Unit,
		Path:     string(key.Path),
	}
}

// symbolsService implements SymbolsService.
type symbolsService struct {
	client *Client
}

var _ SymbolsService = &symbolsService{}

// Symbol is a code symbol returned by the Sourcegraph API.
type Symbol struct {
	graph.Symbol

	Stat graph.Stats `json:",omitempty"`

	DocHTML  string           `json:",omitempty"`
	DefHTML  template.HTML    `json:",omitempty"`
	DocPages []*graph.DocPage `json:",omitempty"`
}

// SymbolSpec returns the SymbolSpec that specifies s.
func (s *Symbol) SymbolSpec() SymbolSpec {
	spec := NewSymbolSpecFromSymbolKey(s.Symbol.SymbolKey)
	spec.SID = int64(s.Symbol.SID)
	return spec
}

func (s *Symbol) XRefs() int { return s.Stat["xrefs"] }
func (s *Symbol) RRefs() int { return s.Stat["rrefs"] }
func (s *Symbol) URefs() int { return s.Stat["urefs"] }

// TotalRefs is the number of unique references of all kinds to s. It
// is computed as (xrefs + rrefs), omitting urefs to avoid double-counting
// references in the same repository.
//
// The number of examples for s is usually TotalRefs() - 1, since the definition
// of a symbol counts as a ref but not an example.
func (s *Symbol) TotalRefs() int { return s.XRefs() + s.RRefs() }

func (s *Symbol) TotalExamples() int { return s.TotalRefs() - 1 }

// SymbolGetOptions specifies options for SymbolsService.Get.
type SymbolGetOptions struct {
	Annotate bool `url:",omitempty"`
	Doc      bool `url:",omitempty"`
	DocPages bool `url:",omitempty"`
}

func (s *symbolsService) Get(symbol SymbolSpec, opt *SymbolGetOptions) (*Symbol, Response, error) {
	var url *url.URL
	var err error
	if symbol.SID != 0 {
		url, err = s.client.url(api_router.SymbolBySID, map[string]string{"SID": fmt.Sprintf("%d", symbol.SID)}, opt)
	} else {
		url, err = s.client.url(api_router.Symbol, symbol.RouteVars(), opt)
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

// SymbolListOptions specifies options for SymbolsService.List.
type SymbolListOptions struct {
	RepositoryURI string `url:",omitempty"`
	Query         string `url:",omitempty"`

	Sort      string `url:",omitempty"`
	Direction string `url:",omitempty"`

	Kinds        []string `url:",omitempty,comma"`
	SpecificKind string   `url:",omitempty"`

	CommitID   string `url:",omitempty"`
	UnitType   string `url:",omitempty"`
	Unit       string `url:",omitempty"`
	Path       string `url:",omitempty"`
	ParentPath string `url:",omitempty"`

	Exported    bool `url:",omitempty"`
	IncludeTest bool `url:",omitempty"`
	Doc         bool `url:",omitempty"`

	ListOptions
}

func (s *symbolsService) List(opt *SymbolListOptions) ([]*Symbol, Response, error) {
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

// SymbolSearchOptions specifies options for SymbolsService.Search
type SymbolSearchOptions struct {
	Query         string
	Exported      bool   `url:",omitempty"` // TODO: make mandatory true for non-repo-limited queries
	RepositoryURI string `url:",omitempty"` // TODO
	ParentPath    string `url:",omitempty"` // TODO
	Instant       bool   `url:",omitempty"`
	ListOptions
}

func (s *symbolsService) Search(opt *SymbolSearchOptions) ([]*Symbol, Response, error) {
	url, err := s.client.url(api_router.SymbolSearch, nil, opt)
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

type SymbolNode struct {
	Name     string
	*Symbol  `json:"Symbol"`
	Children []*SymbolNode
}

type SymbolTreeOptions struct {
	RepositoryURI string `url:",omitempty"`
	// TODO(sqs): kinds' "comma" tag is not respected by gorilla/schema
	Kinds       []string `url:",omitempty,comma"`
	CommitID    string   `url:",omitempty"`
	UnitType    string   `url:",omitempty"`
	Unit        string   `url:",omitempty"`
	Path        string   `url:",omitempty"`
	ParentPath  string   `url:",omitempty"`
	Exported    bool     `url:",omitempty"`
	IncludeTest bool     `url:",omitempty"`

	// TrimRootStubs is whether to eliminate root nodes that have no siblings
	// and only contain children. If true, this operation is applied repeatedly
	// until the root nodes are not subject to trimming.
	TrimRootStubs bool `url:",omitempty"`

	// CollapseUnitDotSymbols is whether to treat a symbol whose path is "." as
	// the symbol for the unit. If false, the tree will contain a parent node
	// for the source unit and a child for the unit's "." symbol (which
	// typically represents the unit).
	CollapseUnitDotSymbols bool `url:",omitempty"`
}

func (s *symbolsService) Tree(opt *SymbolTreeOptions) ([]*SymbolNode, Response, error) {
	url, err := s.client.url(api_router.SymbolsTree, nil, opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var symbolTree []*SymbolNode
	resp, err := s.client.Do(req, &symbolTree)
	if err != nil {
		return nil, resp, err
	}

	return symbolTree, resp, nil
}

// Example is a usage example of a symbol.
type Example struct {
	graph.Ref
	SrcHTML template.HTML
}

type Examples []*Example

func (r *Example) sortKey() string     { return fmt.Sprintf("%+v", r) }
func (vs Examples) Len() int           { return len(vs) }
func (vs Examples) Swap(i, j int)      { vs[i], vs[j] = vs[j], vs[i] }
func (vs Examples) Less(i, j int) bool { return vs[i].sortKey() < vs[j].sortKey() }

// SymbolListExamplesOptions specifies options for SymbolsService.ListExamples.
type SymbolListExamplesOptions struct {
	Annotate bool

	ListOptions
}

func (s *symbolsService) ListExamples(symbol SymbolSpec, opt *SymbolListExamplesOptions) ([]*Example, Response, error) {
	url, err := s.client.url(api_router.SymbolExamples, symbol.RouteVars(), opt)
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

// SymbolListAuthorsOptions specifies options for SymbolsService.ListAuthors.
type SymbolListAuthorsOptions struct {
	ListOptions
}

func (s *symbolsService) ListAuthors(symbol SymbolSpec, opt *SymbolListAuthorsOptions) ([]*AugmentedSymbolAuthor, Response, error) {
	url, err := s.client.url(api_router.SymbolAuthors, symbol.RouteVars(), opt)
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

// SymbolListClientsOptions specifies options for SymbolsService.ListClients.
type SymbolListClientsOptions struct {
	ListOptions
}

func (s *symbolsService) ListClients(symbol SymbolSpec, opt *SymbolListClientsOptions) ([]*AugmentedSymbolClient, Response, error) {
	url, err := s.client.url(api_router.SymbolClients, symbol.RouteVars(), opt)
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

type SymbolDependent struct {
	FromRepo repo.URI `db:"from_repo"`
	Count    int
}

type AugmentedSymbolDependent struct {
	Repo *repo.Repository
	*SymbolDependent
}

// SymbolListDependentsOptions specifies options for SymbolsService.ListDependents.
type SymbolListDependentsOptions struct {
	ListOptions
}

func (s *symbolsService) ListDependents(symbol SymbolSpec, opt *SymbolListDependentsOptions) ([]*AugmentedSymbolDependent, Response, error) {
	url, err := s.client.url(api_router.SymbolDependents, symbol.RouteVars(), opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var dependents []*AugmentedSymbolDependent
	resp, err := s.client.Do(req, &dependents)
	if err != nil {
		return nil, resp, err
	}

	return dependents, resp, nil
}

// SymbolListImplementationsOptions specifies options for
// SymbolsService.ListImplementations.
type SymbolListImplementationsOptions struct {
	ListOptions
}

func (s *symbolsService) ListImplementations(symbol SymbolSpec, opt *SymbolListImplementationsOptions) ([]*Symbol, Response, error) {
	url, err := s.client.url(api_router.SymbolImplementations, symbol.RouteVars(), opt)
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

// SymbolListInterfacesOptions specifies options for
// SymbolsService.ListInterfaces.
type SymbolListInterfacesOptions struct {
	ListOptions
}

func (s *symbolsService) ListInterfaces(symbol SymbolSpec, opt *SymbolListInterfacesOptions) ([]*Symbol, Response, error) {
	url, err := s.client.url(api_router.SymbolInterfaces, symbol.RouteVars(), opt)
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

func (s *symbolsService) CountByRepository(repo RepositorySpec) (*graph.SymbolCounts, Response, error) {
	url, err := s.client.url(api_router.RepositorySymbolCounts, repo.RouteVars(), nil)
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
	Get_                 func(symbol SymbolSpec, opt *SymbolGetOptions) (*Symbol, Response, error)
	List_                func(opt *SymbolListOptions) ([]*Symbol, Response, error)
	Search_              func(opt *SymbolSearchOptions) ([]*Symbol, Response, error)
	Tree_                func(opt *SymbolTreeOptions) ([]*SymbolNode, Response, error)
	ListExamples_        func(symbol SymbolSpec, opt *SymbolListExamplesOptions) ([]*Example, Response, error)
	ListAuthors_         func(symbol SymbolSpec, opt *SymbolListAuthorsOptions) ([]*AugmentedSymbolAuthor, Response, error)
	ListClients_         func(symbol SymbolSpec, opt *SymbolListClientsOptions) ([]*AugmentedSymbolClient, Response, error)
	ListDependents_      func(symbol SymbolSpec, opt *SymbolListDependentsOptions) ([]*AugmentedSymbolDependent, Response, error)
	ListImplementations_ func(symbol SymbolSpec, opt *SymbolListImplementationsOptions) ([]*Symbol, Response, error)
	ListInterfaces_      func(symbol SymbolSpec, opt *SymbolListInterfacesOptions) ([]*Symbol, Response, error)
	CountByRepository_   func(repo RepositorySpec) (*graph.SymbolCounts, Response, error)
}

var _ SymbolsService = MockSymbolsService{}

func (s MockSymbolsService) Get(symbol SymbolSpec, opt *SymbolGetOptions) (*Symbol, Response, error) {
	if s.Get_ == nil {
		return nil, &HTTPResponse{}, nil
	}
	return s.Get_(symbol, opt)
}

func (s MockSymbolsService) List(opt *SymbolListOptions) ([]*Symbol, Response, error) {
	if s.List_ == nil {
		return nil, &HTTPResponse{}, nil
	}
	return s.List_(opt)
}

func (s MockSymbolsService) Search(opt *SymbolSearchOptions) ([]*Symbol, Response, error) {
	if s.Search_ == nil {
		return nil, &HTTPResponse{}, nil
	}
	return s.Search_(opt)
}

func (s MockSymbolsService) Tree(opt *SymbolTreeOptions) ([]*SymbolNode, Response, error) {
	if s.Tree_ == nil {
		return nil, &HTTPResponse{}, nil
	}
	return s.Tree_(opt)
}

func (s MockSymbolsService) ListExamples(symbol SymbolSpec, opt *SymbolListExamplesOptions) ([]*Example, Response, error) {
	if s.ListExamples_ == nil {
		return nil, &HTTPResponse{}, nil
	}
	return s.ListExamples_(symbol, opt)
}

func (s MockSymbolsService) ListAuthors(symbol SymbolSpec, opt *SymbolListAuthorsOptions) ([]*AugmentedSymbolAuthor, Response, error) {
	if s.ListAuthors_ == nil {
		return nil, &HTTPResponse{}, nil
	}
	return s.ListAuthors_(symbol, opt)
}

func (s MockSymbolsService) ListClients(symbol SymbolSpec, opt *SymbolListClientsOptions) ([]*AugmentedSymbolClient, Response, error) {
	if s.ListClients_ == nil {
		return nil, &HTTPResponse{}, nil
	}
	return s.ListClients_(symbol, opt)
}

func (s MockSymbolsService) ListDependents(symbol SymbolSpec, opt *SymbolListDependentsOptions) ([]*AugmentedSymbolDependent, Response, error) {
	if s.ListDependents_ == nil {
		return nil, &HTTPResponse{}, nil
	}
	return s.ListDependents_(symbol, opt)
}

func (s MockSymbolsService) ListImplementations(symbol SymbolSpec, opt *SymbolListImplementationsOptions) ([]*Symbol, Response, error) {
	if s.ListImplementations_ == nil {
		return nil, &HTTPResponse{}, nil
	}
	return s.ListImplementations_(symbol, opt)
}

func (s MockSymbolsService) ListInterfaces(symbol SymbolSpec, opt *SymbolListInterfacesOptions) ([]*Symbol, Response, error) {
	if s.ListInterfaces_ == nil {
		return nil, &HTTPResponse{}, nil
	}
	return s.ListInterfaces_(symbol, opt)
}

func (s MockSymbolsService) CountByRepository(repo RepositorySpec) (*graph.SymbolCounts, Response, error) {
	if s.CountByRepository_ == nil {
		return &graph.SymbolCounts{}, &HTTPResponse{}, nil
	}
	return s.CountByRepository_(repo)
}
