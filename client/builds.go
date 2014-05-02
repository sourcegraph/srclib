package client

import (
	"errors"
	"fmt"
	"time"

	"sourcegraph.com/sourcegraph/api_router"
	"sourcegraph.com/sourcegraph/srcgraph/db_common"
	"sourcegraph.com/sourcegraph/srcgraph/repo"
)

// BuildsService communicates with the build-related endpoints in the
// Sourcegraph API.
type BuildsService interface {
	// Get fetches a build.
	Get(build BuildSpec, opt *BuildGetOptions) (*Build, Response, error)

	// List builds.
	List(opt *BuildListOptions) ([]*Build, Response, error)

	// ListByRepository lists builds for a repository.
	ListByRepository(repo RepositorySpec, opt *BuildListByRepositoryOptions) ([]*Build, Response, error)

	// Create a new build. The build will run asynchronously (Create does not
	// wait for it to return. To monitor the build's status, use Get.)
	Create(repo RepositorySpec, conf BuildConfig) (*Build, Response, error)
}

type buildsService struct {
	client *Client
}

var _ BuildsService = &buildsService{}

type BuildSpec struct {
	Repo RepositorySpec
	BID  int64
}

func (s *BuildSpec) RouteVars() map[string]string {
	return map[string]string{"RepoURI": s.Repo.URI, "BID": fmt.Sprintf("%d", s.BID)}
}

// A Build represents a scheduled, completed, or failed repository analysis and
// import job.
type Build struct {
	BID       int64
	Repo      repo.RID
	CommitID  string             `db:"commit_id"`
	CreatedAt time.Time          `db:"created_at"`
	StartedAt db_common.NullTime `db:"started_at"`
	EndedAt   db_common.NullTime `db:"ended_at"`
	Success   bool
	Failure   bool

	BuildConfig
}

// BuildConfig configures a repository build.
type BuildConfig struct {
	// Import is whether to import the build data into the database when the
	// build is complete. The data must be imported for Sourcegraph's web app or
	// API to use it, except that unimported build data is available through the
	// BuildData service. (TODO(sqs): BuildData isn't yet implemented.)
	Import bool

	// Queue is whether this build should be enqueued. If enqueued, any worker
	// may begin running this build. If not enqueued, it is up to the client to
	// run the build and update it accordingly.
	Queue bool
}

var ErrBuildNotFound = errors.New("build not found")

type BuildGetOptions struct{}

func (s *buildsService) Get(build BuildSpec, opt *BuildGetOptions) (*Build, Response, error) {
	url, err := s.client.url(api_router.RepositoryBuild, build.RouteVars(), opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var build_ *Build
	resp, err := s.client.Do(req, &build_)
	if err != nil {
		return nil, resp, err
	}

	return build_, nil, nil
}

type BuildListOptions struct {
	Ended     bool `url:",omitempty"`
	Succeeded bool `url:",omitempty"`

	ListOptions
}

func (s *buildsService) List(opt *BuildListOptions) ([]*Build, Response, error) {
	url, err := s.client.url(api_router.Builds, nil, opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var builds []*Build
	resp, err := s.client.Do(req, &builds)
	if err != nil {
		return nil, resp, err
	}

	return builds, resp, nil
}

type BuildListByRepositoryOptions struct {
	BuildListOptions
	Rev string `url:",omitempty"`
}

func (s *buildsService) ListByRepository(repo RepositorySpec, opt *BuildListByRepositoryOptions) ([]*Build, Response, error) {
	url, err := s.client.url(api_router.RepositoryBuilds, map[string]string{"RepoURI": repo.URI}, opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var builds []*Build
	resp, err := s.client.Do(req, &builds)
	if err != nil {
		return nil, resp, err
	}

	return builds, resp, nil
}

func (s *buildsService) Create(repo RepositorySpec, conf BuildConfig) (*Build, Response, error) {
	url, err := s.client.url(api_router.RepositoryBuildsCreate, map[string]string{"RepoURI": repo.URI}, nil)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("POST", url.String(), conf)
	if err != nil {
		return nil, nil, err
	}

	var build *Build
	resp, err := s.client.Do(req, &build)
	if err != nil {
		return nil, resp, err
	}

	return build, resp, nil
}

type MockBuildsService struct {
	Get_              func(build BuildSpec, opt *BuildGetOptions) (*Build, Response, error)
	List_             func(opt *BuildListOptions) ([]*Build, Response, error)
	ListByRepository_ func(repo RepositorySpec, opt *BuildListByRepositoryOptions) ([]*Build, Response, error)
	Create_           func(repo RepositorySpec, conf BuildConfig) (*Build, Response, error)
}

var _ BuildsService = MockBuildsService{}

func (s MockBuildsService) Get(build BuildSpec, opt *BuildGetOptions) (*Build, Response, error) {
	if s.Get_ == nil {
		return nil, &HTTPResponse{}, nil
	}
	return s.Get_(build, opt)
}

func (s MockBuildsService) List(opt *BuildListOptions) ([]*Build, Response, error) {
	if s.List_ == nil {
		return nil, nil, nil
	}
	return s.List_(opt)
}

func (s MockBuildsService) ListByRepository(repo RepositorySpec, opt *BuildListByRepositoryOptions) ([]*Build, Response, error) {
	if s.ListByRepository_ == nil {
		return nil, &HTTPResponse{}, nil
	}
	return s.ListByRepository_(repo, opt)
}

func (s MockBuildsService) Create(repo RepositorySpec, conf BuildConfig) (*Build, Response, error) {
	if s.Create_ == nil {
		return nil, nil, nil
	}
	return s.Create_(repo, conf)
}
