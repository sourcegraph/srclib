package client

import (
	"fmt"
	"text/template"

	"sourcegraph.com/sourcegraph/api_router"
	"sourcegraph.com/sourcegraph/srcgraph/authorship"
	"sourcegraph.com/sourcegraph/srcgraph/person"
	"sourcegraph.com/sourcegraph/srcgraph/repo"
)

// RepositoriesService communicates with the repository-related endpoints in the
// Sourcegraph API.
type RepositoriesService interface {
	// Get fetches a repository.
	Get(repo RepositorySpec) (*Repository, Response, error)

	// GetReadme fetches the formatted README file for a repository.
	GetReadme(repo RepositorySpec) (string, Response, error)

	// List repositories.
	List(opt *RepositoryListOptions) ([]*repo.Repository, Response, error)

	// ListBadges lists the available badges for repo.
	ListBadges(repo RepositorySpec) ([]*Badge, Response, error)

	// ListCounters lists the available counters for repo.
	ListCounters(repo RepositorySpec) ([]*Counter, Response, error)

	// ListAuthors lists people who have contributed (i.e., committed) code to
	// repo.
	ListAuthors(repo RepositorySpec, opt *RepositoryListAuthorsOptions) ([]*AugmentedRepoAuthor, Response, error)

	// ListClients lists people who reference symbols defined in repo.
	ListClients(repo RepositorySpec, opt *RepositoryListClientsOptions) ([]*AugmentedRepoClient, Response, error)

	// ListDependents lists repositories that contain symbols referenced by
	// repo.
	ListDependencies(repo RepositorySpec, opt *RepositoryListDependenciesOptions) ([]*AugmentedRepoDependency, Response, error)

	// ListDependents lists repositories that reference symbols defined in repo.
	ListDependents(repo RepositorySpec, opt *RepositoryListDependentsOptions) ([]*AugmentedRepoDependent, Response, error)

	// ListByOwner lists repositories owned by person. Currently only GitHub
	// repositories have an owner (e.g., alice owns github.com/alice/foo).
	ListByOwner(person PersonSpec, opt *RepositoryListByOwnerOptions) ([]*repo.Repository, Response, error)

	// ListByContributor lists repositories that person has contributed (i.e.,
	// committed) code to.
	ListByContributor(person PersonSpec, opt *RepositoryListByContributorOptions) ([]*AugmentedRepoContribution, Response, error)

	// ListByClient lists repositories that contain symbols referenced by
	// person.
	ListByClient(person PersonSpec, opt *RepositoryListByClientOptions) ([]*AugmentedRepoUsageByClient, Response, error)

	// ListByRefdAuthor lists repositories that reference code authored by
	// person.
	ListByRefdAuthor(person PersonSpec, opt *RepositoryListByRefdAuthorOptions) ([]*AugmentedRepoUsageOfAuthor, Response, error)
}

// repositoriesService implements RepositoriesService.
type repositoriesService struct {
	client *Client
}

var _ RepositoriesService = &repositoriesService{}

// RepositorySpec specifies a repository.
type RepositorySpec struct {
	URI      string
	CommitID string
}

func (s RepositorySpec) String() string { return s.URI }

// Repository is a code repository returned by the Sourcegraph API.
type Repository struct {
	*repo.Repository

	Stat repo.Stats `json:",omitempty"`

	// Unsupported is whether Sourcegraph doesn't support this repository.
	Unsupported bool `json:",omitempty"`

	NoticeTitle, NoticeBody string `json:",omitempty"`
}

// Spec returns the RepositorySpec that specifies r.
func (r *Repository) Spec() RepositorySpec { return RepositorySpec{URI: string(r.Repository.URI)} }

func (s *repositoriesService) Get(repo RepositorySpec) (*Repository, Response, error) {
	url, err := s.client.url(api_router.Repository, map[string]string{"RepoURI": repo.URI}, nil)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var repo_ *Repository
	resp, err := s.client.Do(req, &repo_)
	if err != nil {
		return nil, resp, err
	}

	return repo_, resp, nil
}

func (s *repositoriesService) GetReadme(repo RepositorySpec) (string, Response, error) {
	url, err := s.client.url(api_router.RepositoryReadme, map[string]string{"RepoURI": repo.URI}, nil)
	if err != nil {
		return "", nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return "", nil, err
	}

	var readme []byte
	resp, err := s.client.Do(req, &readme)
	if err != nil {
		return "", resp, err
	}

	return string(readme), resp, nil
}

type RepositoryListOptions struct {
	URIs  []string `url:",comma,omitempty"`
	Query string   `url:",omitempty"`

	Sort      string `url:",omitempty"`
	Direction string `url:",omitempty"`

	NoFork bool `url:",omitempty"`

	ListOptions
}

func (s *repositoriesService) List(opt *RepositoryListOptions) ([]*repo.Repository, Response, error) {
	url, err := s.client.url(api_router.Repositories, nil, opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var repos []*repo.Repository
	resp, err := s.client.Do(req, &repos)
	if err != nil {
		return nil, resp, err
	}

	return repos, resp, nil
}

type Badge struct {
	Name              string
	Description       string
	ImageURL          string
	UncountedImageURL string
	Markdown          string
}

func (b *Badge) HTML() string {
	return fmt.Sprintf(`<img src="%s" alt="%s">`, template.HTMLEscapeString(b.ImageURL), template.HTMLEscapeString(b.Name))
}

func (s *repositoriesService) ListBadges(repo RepositorySpec) ([]*Badge, Response, error) {
	url, err := s.client.url(api_router.RepositoryBadges, map[string]string{"RepoURI": repo.URI}, nil)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var badges []*Badge
	resp, err := s.client.Do(req, &badges)
	if err != nil {
		return nil, resp, err
	}

	return badges, resp, nil
}

type Counter struct {
	Name              string
	Description       string
	ImageURL          string
	UncountedImageURL string
	Markdown          string
}

func (c *Counter) HTML() string {
	return fmt.Sprintf(`<img src="%s" alt="%s">`, template.HTMLEscapeString(c.ImageURL), template.HTMLEscapeString(c.Name))
}

func (s *repositoriesService) ListCounters(repo RepositorySpec) ([]*Counter, Response, error) {
	url, err := s.client.url(api_router.RepositoryCounters, map[string]string{"RepoURI": repo.URI}, nil)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var counters []*Counter
	resp, err := s.client.Do(req, &counters)
	if err != nil {
		return nil, resp, err
	}

	return counters, resp, nil
}

// AugmentedRepoAuthor is a rel.RepoAuthor with the full person.User and
// graph.Symbol structs embedded.
type AugmentedRepoAuthor struct {
	User *person.User
	*authorship.RepoAuthor
}

type RepositoryListAuthorsOptions struct {
	ListOptions
}

func (s *repositoriesService) ListAuthors(repo RepositorySpec, opt *RepositoryListAuthorsOptions) ([]*AugmentedRepoAuthor, Response, error) {
	url, err := s.client.url(api_router.RepositoryAuthors, map[string]string{"RepoURI": repo.URI}, opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var authors []*AugmentedRepoAuthor
	resp, err := s.client.Do(req, &authors)
	if err != nil {
		return nil, resp, err
	}

	return authors, resp, nil
}

// AugmentedRepoClient is a rel.RepoClient with the full person.User and
// graph.Symbol structs embedded.
type AugmentedRepoClient struct {
	User *person.User
	*authorship.RepoClient
}

type RepositoryListClientsOptions struct {
	ListOptions
}

func (s *repositoriesService) ListClients(repo RepositorySpec, opt *RepositoryListClientsOptions) ([]*AugmentedRepoClient, Response, error) {
	url, err := s.client.url(api_router.RepositoryClients, map[string]string{"RepoURI": repo.URI}, opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var clients []*AugmentedRepoClient
	resp, err := s.client.Do(req, &clients)
	if err != nil {
		return nil, resp, err
	}

	return clients, resp, nil
}

type RepoDependency struct {
	ToRepo repo.URI `db:"to_repo"`
}

type AugmentedRepoDependency struct {
	Repo *repo.Repository
	*RepoDependency
}

type RepositoryListDependenciesOptions struct {
	ListOptions
}

func (s *repositoriesService) ListDependencies(repo RepositorySpec, opt *RepositoryListDependenciesOptions) ([]*AugmentedRepoDependency, Response, error) {
	url, err := s.client.url(api_router.RepositoryDependencies, map[string]string{"RepoURI": repo.URI}, opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var dependencies []*AugmentedRepoDependency
	resp, err := s.client.Do(req, &dependencies)
	if err != nil {
		return nil, resp, err
	}

	return dependencies, resp, nil
}

type RepoDependent struct {
	FromRepo repo.URI `db:"from_repo"`
}

type AugmentedRepoDependent struct {
	Repo *repo.Repository
	*RepoDependent
}

type RepositoryListDependentsOptions struct{ ListOptions }

func (s *repositoriesService) ListDependents(repo RepositorySpec, opt *RepositoryListDependentsOptions) ([]*AugmentedRepoDependent, Response, error) {
	url, err := s.client.url(api_router.RepositoryDependents, map[string]string{"RepoURI": repo.URI}, opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var dependents []*AugmentedRepoDependent
	resp, err := s.client.Do(req, &dependents)
	if err != nil {
		return nil, resp, err
	}

	return dependents, resp, nil
}

type AugmentedRepoContribution struct {
	Repo *repo.Repository
	*authorship.RepoContribution
}

type RepositoryListByOwnerOptions struct {
	ListOptions
}

func (s *repositoriesService) ListByOwner(person PersonSpec, opt *RepositoryListByOwnerOptions) ([]*repo.Repository, Response, error) {
	url, err := s.client.url(api_router.PersonOwnedRepositories, map[string]string{"PersonSpec": person.PathComponent()}, opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var repos []*repo.Repository
	resp, err := s.client.Do(req, &repos)
	if err != nil {
		return nil, resp, err
	}

	return repos, resp, nil
}

type RepositoryListByContributorOptions struct {
	NoFork bool
	ListOptions
}

func (s *repositoriesService) ListByContributor(person PersonSpec, opt *RepositoryListByContributorOptions) ([]*AugmentedRepoContribution, Response, error) {
	url, err := s.client.url(api_router.PersonRepositoryContributions, map[string]string{"PersonSpec": person.PathComponent()}, opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var repos []*AugmentedRepoContribution
	resp, err := s.client.Do(req, &repos)
	if err != nil {
		return nil, resp, err
	}

	return repos, resp, nil
}

// AugmentedRepoUsageByClient is a authorship.RepoUsageByClient with the full repo.Repository
// struct embedded.
type AugmentedRepoUsageByClient struct {
	SymbolRepo *repo.Repository
	*authorship.RepoUsageByClient
}

type RepositoryListByClientOptions struct {
	ListOptions
}

func (s *repositoriesService) ListByClient(person PersonSpec, opt *RepositoryListByClientOptions) ([]*AugmentedRepoUsageByClient, Response, error) {
	url, err := s.client.url(api_router.PersonRepositoryDependencies, map[string]string{"PersonSpec": person.PathComponent()}, opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var repos []*AugmentedRepoUsageByClient
	resp, err := s.client.Do(req, &repos)
	if err != nil {
		return nil, resp, err
	}

	return repos, resp, nil
}

// AugmentedRepoUsageOfAuthor is a authorship.RepoUsageOfAuthor with the full
// repo.Repository struct embedded.
type AugmentedRepoUsageOfAuthor struct {
	Repo *repo.Repository
	*authorship.RepoUsageOfAuthor
}

type RepositoryListByRefdAuthorOptions struct {
	ListOptions
}

func (s *repositoriesService) ListByRefdAuthor(person PersonSpec, opt *RepositoryListByRefdAuthorOptions) ([]*AugmentedRepoUsageOfAuthor, Response, error) {
	url, err := s.client.url(api_router.PersonRepositoryDependents, map[string]string{"PersonSpec": person.PathComponent()}, opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var repos []*AugmentedRepoUsageOfAuthor
	resp, err := s.client.Do(req, &repos)
	if err != nil {
		return nil, resp, err
	}

	return repos, resp, nil
}

type MockRepositoriesService struct {
	Get_               func(spec RepositorySpec) (*Repository, Response, error)
	GetReadme_         func(repo RepositorySpec) (string, Response, error)
	List_              func(opt *RepositoryListOptions) ([]*repo.Repository, Response, error)
	ListBadges_        func(repo RepositorySpec) ([]*Badge, Response, error)
	ListCounters_      func(repo RepositorySpec) ([]*Counter, Response, error)
	ListAuthors_       func(repo RepositorySpec, opt *RepositoryListAuthorsOptions) ([]*AugmentedRepoAuthor, Response, error)
	ListClients_       func(repo RepositorySpec, opt *RepositoryListClientsOptions) ([]*AugmentedRepoClient, Response, error)
	ListDependencies_  func(repo RepositorySpec, opt *RepositoryListDependenciesOptions) ([]*AugmentedRepoDependency, Response, error)
	ListDependents_    func(repo RepositorySpec, opt *RepositoryListDependentsOptions) ([]*AugmentedRepoDependent, Response, error)
	ListByOwner_       func(person PersonSpec, opt *RepositoryListByOwnerOptions) ([]*repo.Repository, Response, error)
	ListByContributor_ func(person PersonSpec, opt *RepositoryListByContributorOptions) ([]*AugmentedRepoContribution, Response, error)
	ListByClient_      func(person PersonSpec, opt *RepositoryListByClientOptions) ([]*AugmentedRepoUsageByClient, Response, error)
	ListByRefdAuthor_  func(person PersonSpec, opt *RepositoryListByRefdAuthorOptions) ([]*AugmentedRepoUsageOfAuthor, Response, error)
}

var _ RepositoriesService = MockRepositoriesService{}

func (s MockRepositoriesService) Get(repo RepositorySpec) (*Repository, Response, error) {
	if s.Get_ == nil {
		return nil, &HTTPResponse{}, nil
	}
	return s.Get_(repo)
}

func (s MockRepositoriesService) GetReadme(repo RepositorySpec) (string, Response, error) {
	if s.GetReadme_ == nil {
		return "", nil, nil
	}
	return s.GetReadme_(repo)
}

func (s MockRepositoriesService) List(opt *RepositoryListOptions) ([]*repo.Repository, Response, error) {
	if s.List_ == nil {
		return nil, &HTTPResponse{}, nil
	}
	return s.List_(opt)
}

func (s MockRepositoriesService) ListBadges(repo RepositorySpec) ([]*Badge, Response, error) {
	if s.ListBadges_ == nil {
		return nil, &HTTPResponse{}, nil
	}
	return s.ListBadges_(repo)
}

func (s MockRepositoriesService) ListCounters(repo RepositorySpec) ([]*Counter, Response, error) {
	if s.ListCounters_ == nil {
		return nil, &HTTPResponse{}, nil
	}
	return s.ListCounters_(repo)
}

func (s MockRepositoriesService) ListAuthors(repo RepositorySpec, opt *RepositoryListAuthorsOptions) ([]*AugmentedRepoAuthor, Response, error) {
	if s.ListAuthors_ == nil {
		return nil, &HTTPResponse{}, nil
	}
	return s.ListAuthors_(repo, opt)
}

func (s MockRepositoriesService) ListClients(repo RepositorySpec, opt *RepositoryListClientsOptions) ([]*AugmentedRepoClient, Response, error) {
	if s.ListClients_ == nil {
		return nil, &HTTPResponse{}, nil
	}
	return s.ListClients_(repo, opt)
}

func (s MockRepositoriesService) ListDependencies(repo RepositorySpec, opt *RepositoryListDependenciesOptions) ([]*AugmentedRepoDependency, Response, error) {
	if s.ListDependencies_ == nil {
		return nil, &HTTPResponse{}, nil
	}
	return s.ListDependencies_(repo, opt)
}

func (s MockRepositoriesService) ListDependents(repo RepositorySpec, opt *RepositoryListDependentsOptions) ([]*AugmentedRepoDependent, Response, error) {
	if s.ListDependents_ == nil {
		return nil, &HTTPResponse{}, nil
	}
	return s.ListDependents_(repo, opt)
}

func (s MockRepositoriesService) ListByOwner(person PersonSpec, opt *RepositoryListByOwnerOptions) ([]*repo.Repository, Response, error) {
	if s.ListByOwner_ == nil {
		return nil, nil, nil
	}
	return s.ListByOwner_(person, opt)
}

func (s MockRepositoriesService) ListByContributor(person PersonSpec, opt *RepositoryListByContributorOptions) ([]*AugmentedRepoContribution, Response, error) {
	if s.ListByContributor_ == nil {
		return nil, nil, nil
	}
	return s.ListByContributor_(person, opt)
}

func (s MockRepositoriesService) ListByClient(person PersonSpec, opt *RepositoryListByClientOptions) ([]*AugmentedRepoUsageByClient, Response, error) {
	if s.ListByClient_ == nil {
		return nil, &HTTPResponse{}, nil
	}
	return s.ListByClient_(person, opt)
}

func (s MockRepositoriesService) ListByRefdAuthor(person PersonSpec, opt *RepositoryListByRefdAuthorOptions) ([]*AugmentedRepoUsageOfAuthor, Response, error) {
	if s.ListByRefdAuthor_ == nil {
		return nil, &HTTPResponse{}, nil
	}
	return s.ListByRefdAuthor_(person, opt)
}
