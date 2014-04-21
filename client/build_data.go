package client

import (
	"io"

	"sourcegraph.com/sourcegraph/api_router"
	"sourcegraph.com/sourcegraph/srcgraph/buildstore"
)

type BuildDataService interface {
	List(repo RepositorySpec, commitID string, opt *BuildDataListOptions) ([]*buildstore.BuildDataFileInfo, *Response, error)
	Get(file BuildDataFileSpec) ([]byte, *Response, error)
	Upload(spec BuildDataFileSpec, body io.Reader) (*Response, error)
}

type buildDataService struct {
	client *Client
}

var _ BuildDataService = &buildDataService{}

type BuildDataFileSpec struct {
	Repo     RepositorySpec
	CommitID string
	Path     string
}

type BuildDataListOptions struct {
	ListOptions
}

func (s *buildDataService) List(repo RepositorySpec, commitID string, opt *BuildDataListOptions) ([]*buildstore.BuildDataFileInfo, *Response, error) {
	url, err := s.client.url(api_router.RepositoryBuildDataIndex, map[string]string{"RepoURI": repo.URI, "CommitID": commitID}, opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var fileInfo []*buildstore.BuildDataFileInfo
	resp, err := s.client.Do(req, &fileInfo)
	if err != nil {
		return nil, resp, err
	}

	return fileInfo, resp, nil
}

func (s *buildDataService) Get(file BuildDataFileSpec) ([]byte, *Response, error) {
	url, err := s.client.url(api_router.RepositoryBuildDataFile, map[string]string{"RepoURI": file.Repo.URI, "CommitID": file.CommitID, "Path": file.Path}, nil)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var data []byte
	resp, err := s.client.Do(req, &data)
	if err != nil {
		return nil, resp, err
	}

	return data, resp, nil
}

func (s *buildDataService) Upload(file BuildDataFileSpec, body io.Reader) (*Response, error) {
	url, err := s.client.url(api_router.RepositoryBuildDataFile, map[string]string{"RepoURI": file.Repo.URI, "CommitID": file.CommitID, "Path": file.Path}, nil)
	if err != nil {
		return nil, err
	}

	req, err := s.client.NewRequest("PUT", url.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(req, nil)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

type MockBuildDataService struct {
	List_   func(repo RepositorySpec, commitID string, opt *BuildDataListOptions) ([]*buildstore.BuildDataFileInfo, *Response, error)
	Get_    func(file BuildDataFileSpec) ([]byte, *Response, error)
	Upload_ func(spec BuildDataFileSpec, body io.Reader) (*Response, error)
}

var _ BuildDataService = MockBuildDataService{}

func (s MockBuildDataService) List(repo RepositorySpec, commitID string, opt *BuildDataListOptions) ([]*buildstore.BuildDataFileInfo, *Response, error) {
	if s.List_ == nil {
		return nil, &Response{}, nil
	}
	return s.List_(repo, commitID, opt)
}

func (s MockBuildDataService) Get(file BuildDataFileSpec) ([]byte, *Response, error) {
	if s.Get_ == nil {
		return nil, &Response{}, nil
	}
	return s.Get_(file)
}

func (s MockBuildDataService) Upload(spec BuildDataFileSpec, body io.Reader) (*Response, error) {
	if s.Upload_ == nil {
		return nil, nil
	}
	return s.Upload_(spec, body)
}
