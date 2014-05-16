package client

import (
	"errors"
	"fmt"
	"html/template"

	"sourcegraph.com/sourcegraph/api_router"
)

type RepositoryTreeService interface {
	Get(entry TreeEntrySpec, opt *RepositoryTreeGetOptions) (*TreeEntry, Response, error)
}

type repositoryTreeService struct {
	client *Client
}

var _ RepositoryTreeService = &repositoryTreeService{}

type TreeEntrySpec struct {
	Repo RepositorySpec
	Path string
}

func (s *TreeEntrySpec) RouteVars() map[string]string {
	return map[string]string{"RepoURI": s.Repo.URI, "Rev": s.Repo.CommitID, "Path": s.Path}
}

func (s TreeEntrySpec) String() string {
	return fmt.Sprintf("%s: %s (rev %q)", s.Repo.URI, s.Path, s.Repo.CommitID)
}

type TreeEntryType string

const (
	File TreeEntryType = "file"
	Dir  TreeEntryType = "dir"
)

type TreeEntry struct {
	Type TreeEntryType

	File *FileData `json:",omitempty"`

	// TODO(sqs): add Dir field
}

type RepositoryTreeGetOptions struct {
	Annotated bool
}

func (s *repositoryTreeService) Get(entry TreeEntrySpec, opt *RepositoryTreeGetOptions) (*TreeEntry, Response, error) {
	if !opt.Annotated {
		return nil, nil, errors.New("non-annotated is not yet supported")
	}

	url, err := s.client.url(api_router.RepositoryTreeEntry, entry.RouteVars(), opt)
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

	e := &TreeEntry{}
	switch resp.Header.Get("content-type") {
	case "application/x-directory":
		e.Type = Dir
		// TODO(sqs): fill in information about this dir
	default:
		e.Type = File
		e.File = &FileData{
			Repo:       entry.Repo,
			File:       entry.Path,
			EntireFile: true,
			Annotated:  template.HTML(data),
		}
	}

	return e, resp, nil
}

type MockRepositoryTreeService struct {
	Get_ func(entry TreeEntrySpec, opt *RepositoryTreeGetOptions) (*TreeEntry, Response, error)
}

var _ RepositoryTreeService = MockRepositoryTreeService{}

func (s MockRepositoryTreeService) Get(entry TreeEntrySpec, opt *RepositoryTreeGetOptions) (*TreeEntry, Response, error) {
	if s.Get_ == nil {
		return nil, &HTTPResponse{}, nil
	}
	return s.Get_(entry, opt)
}
