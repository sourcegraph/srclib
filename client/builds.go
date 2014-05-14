package client

import (
	"errors"
	"fmt"
	"time"

	"strconv"

	"sourcegraph.com/sourcegraph/api_router"
	"sourcegraph.com/sourcegraph/srcgraph/db_common"
	"sourcegraph.com/sourcegraph/srcgraph/repo"
	"sourcegraph.com/sourcegraph/srcgraph/task2"
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

	// ListBuildTasks lists the tasks associated with a build.
	ListBuildTasks(build BuildSpec, opt *BuildTaskListOptions) ([]*BuildTask, Response, error)
}

type buildsService struct {
	client *Client
}

var _ BuildsService = &buildsService{}

type BuildSpec struct {
	BID int64
}

func (s *BuildSpec) RouteVars() map[string]string {
	return map[string]string{"BID": fmt.Sprintf("%d", s.BID)}
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

	// RepoURI is populated (as a convenience) in results by Get and List but
	// should not be set when creating builds (it will be ignored).
	RepoURI repo.URI `db:"repo_uri" json:",omitempty"`
}

// IDString returns a succinct string that uniquely identifies this build.
func (b *Build) IDString() string { return buildIDString(b.BID) }

func buildIDString(bid int64) string { return "B" + strconv.FormatInt(bid, 36) }

// A BuildTask represents an individual step of a build.
type BuildTask struct {
	TID int64

	// BID is the build that this task is a part of.
	BID int64

	UnitType string
	Unit     string

	Title string

	StartedAt db_common.NullTime `db:"started_at"`
	EndedAt   db_common.NullTime `db:"ended_at"`

	Success bool
	Failure bool
}

// IDString returns a succinct string that uniquely identifies this build task.
func (t *BuildTask) IDString() string {
	return buildIDString(t.BID) + "-T" + strconv.FormatInt(t.TID, 36)
}

// LogURL is the URL to the logs for this task.
func (t *BuildTask) LogURL() string {
	return task2.LogURLForTag(t.IDString())
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
	url, err := s.client.url(api_router.Build, build.RouteVars(), opt)
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
	Queued    bool `url:",omitempty"`
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
	url, err := s.client.url(api_router.RepositoryBuildsCreate, repo.RouteVars(), nil)
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

type BuildTaskListOptions struct{ ListOptions }

func (s *buildsService) ListBuildTasks(build BuildSpec, opt *BuildTaskListOptions) ([]*BuildTask, Response, error) {
	url, err := s.client.url(api_router.BuildTasks, build.RouteVars(), opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var tasks []*BuildTask
	resp, err := s.client.Do(req, &tasks)
	if err != nil {
		return nil, resp, err
	}

	return tasks, resp, nil
}

type MockBuildsService struct {
	Get_              func(build BuildSpec, opt *BuildGetOptions) (*Build, Response, error)
	List_             func(opt *BuildListOptions) ([]*Build, Response, error)
	ListByRepository_ func(repo RepositorySpec, opt *BuildListByRepositoryOptions) ([]*Build, Response, error)
	Create_           func(repo RepositorySpec, conf BuildConfig) (*Build, Response, error)
	ListBuildTasks_   func(build BuildSpec, opt *BuildTaskListOptions) ([]*BuildTask, Response, error)
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

func (s MockBuildsService) ListBuildTasks(build BuildSpec, opt *BuildTaskListOptions) ([]*BuildTask, Response, error) {
	if s.ListBuildTasks_ == nil {
		return nil, nil, nil
	}
	return s.ListBuildTasks_(build, opt)
}
