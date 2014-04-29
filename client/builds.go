package client

import (
	"errors"
	"fmt"
	"time"

	"sourcegraph.com/sourcegraph/api_router"
	"sourcegraph.com/sourcegraph/srcgraph/db_common"
	"sourcegraph.com/sourcegraph/srcgraph/repo"
)

type BuildsService interface {
	Get(build BuildSpec, opt *BuildGetOptions) (*Build, Response, error)
	ListByRepository(repo RepositorySpec, opt *BuildListByRepositoryOptions) ([]*Build, Response, error)
}

type buildsService struct {
	client *Client
}

var _ BuildsService = &buildsService{}

type BuildSpec struct {
	Repo RepositorySpec
	BID  int64
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
}

var ErrBuildNotFound = errors.New("build not found")

type BuildGetOptions struct{}

func (s *buildsService) Get(build BuildSpec, opt *BuildGetOptions) (*Build, Response, error) {
	url, err := s.client.url(api_router.RepositoryBuild, map[string]string{"RepoURI": build.Repo.URI, "BID": fmt.Sprintf("%d", build.BID)}, opt)
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

type BuildListByRepositoryOptions struct{ ListOptions }

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

type MockBuildsService struct {
	Get_              func(build BuildSpec, opt *BuildGetOptions) (*Build, Response, error)
	ListByRepository_ func(repo RepositorySpec, opt *BuildListByRepositoryOptions) ([]*Build, Response, error)
}

var _ BuildsService = MockBuildsService{}

func (s MockBuildsService) Get(build BuildSpec, opt *BuildGetOptions) (*Build, Response, error) {
	if s.Get_ == nil {
		return nil, &HTTPResponse{}, nil
	}
	return s.Get_(build, opt)
}
func (s MockBuildsService) ListByRepository(repo RepositorySpec, opt *BuildListByRepositoryOptions) ([]*Build, Response, error) {
	if s.ListByRepository_ == nil {
		return nil, &HTTPResponse{}, nil
	}
	return s.ListByRepository_(repo, opt)
}
