package client

import (
	"errors"
	"fmt"
	"html/template"

	"sourcegraph.com/sourcegraph/api_router"
)

type RepositoryTreeService interface {
	Get(entry TreeEntrySpec, opt *RepositoryTreeGetOptions) (*TreeEntry, *Response, error)
}

type repositoryTreeService struct {
	client *Client
}

var _ RepositoryTreeService = &repositoryTreeService{}

type TreeEntrySpec struct {
	Repo RepositorySpec
	Rev  string
	Path string
}

func (s TreeEntrySpec) String() string {
	return fmt.Sprintf("%s: %s (rev %q)", s.Repo, s.Path, s.Rev)
}

type TreeEntryType string

const (
	File TreeEntryType = "file"
	Dir  TreeEntryType = "dir"
)

type TreeEntry struct {
	Type TreeEntryType

	// TODO(sqs): why is this HTML?
	Data template.HTML
}

type RepositoryTreeGetOptions struct {
	Annotated bool
}

func (s *repositoryTreeService) Get(entry TreeEntrySpec, opt *RepositoryTreeGetOptions) (*TreeEntry, *Response, error) {
	if !opt.Annotated {
		return nil, nil, errors.New("non-annotated is not yet supported")
	}

	url, err := s.client.url(api_router.RepositoryTreeEntry, map[string]string{"RepoURI": entry.Repo.URI, "Rev": entry.Rev, "Path": entry.Path}, opt)
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

	filetypename := resp.Header.Get("Content-Type")
	filetype := File
	if filetypename == "application/x-directory" {
		filetype = Dir
	}
	return &TreeEntry{Data: template.HTML(data), Type: filetype}, resp, nil
}

type MockRepositoryTreeService struct {
	Get_ func(entry TreeEntrySpec, opt *RepositoryTreeGetOptions) (*TreeEntry, *Response, error)
}

var _ RepositoryTreeService = MockRepositoryTreeService{}

func (s MockRepositoryTreeService) Get(entry TreeEntrySpec, opt *RepositoryTreeGetOptions) (*TreeEntry, *Response, error) {
	if s.Get_ == nil {
		return nil, &Response{}, nil
	}
	return s.Get_(entry, opt)
}
