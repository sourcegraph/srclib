package client

import (
	"fmt"

	"sourcegraph.com/sourcegraph/api_router"
	"sourcegraph.com/sourcegraph/build_db"
)

type Build build_db.Build

type BuildsService interface {
	Get(build BuildSpec, opt *BuildGetOptions) (*Build, *Response, error)
	ListByRepository(repo RepositorySpec, opt *RepositoryBuildListOptions) ([]*Build, *Response, error)
}

type buildsService struct {
	client *Client
}

var _ BuildsService = &buildsService{}

type BuildSpec struct {
	Repo RepositorySpec
	BID  int64
}

type BuildGetOptions struct{}

func (s *buildsService) Get(build BuildSpec, opt *BuildGetOptions) (*Build, *Response, error) {
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

type RepositoryBuildListOptions struct{}

func (s *buildsService) ListByRepository(repo RepositorySpec, opt *RepositoryBuildListOptions) ([]*Build, *Response, error) {
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
	Get_              func(build BuildSpec, opt *BuildGetOptions) (*Build, *Response, error)
	ListByRepository_ func(repo RepositorySpec, opt *RepositoryBuildListOptions) ([]*Build, *Response, error)
}

var _ BuildsService = MockBuildsService{}

func (s MockBuildsService) Get(build BuildSpec, opt *BuildGetOptions) (*Build, *Response, error) {
	if s.Get_ == nil {
		return nil, &Response{}, nil
	}
	return s.Get_(build, opt)
}
func (s MockBuildsService) ListByRepository(repo RepositorySpec, opt *RepositoryBuildListOptions) ([]*Build, *Response, error) {
	if s.ListByRepository_ == nil {
		return nil, &Response{}, nil
	}
	return s.ListByRepository_(repo, opt)
}
