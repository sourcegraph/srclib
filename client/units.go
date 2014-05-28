package client

import (
	"sourcegraph.com/sourcegraph/api_router"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
)

// UnitsService communicates with the source unit-related endpoints in
// the Sourcegraph API.
type UnitsService interface {
	// List units.
	List(opt *UnitListOptions) ([]*unit.RepoSourceUnit, Response, error)
}

// UnitSpec specifies a source unit.
type UnitSpec struct {
	Repo     string
	CommitID string
	UnitType string
	Unit     string
}

func (s *UnitSpec) RouteVars() map[string]string {
	m := map[string]string{"RepoURI": s.Repo, "UnitType": s.UnitType, "Unit": s.Unit}
	if s.CommitID != "" {
		m["Rev"] = s.CommitID
	}
	return m
}

// unitsService implements UnitsService.
type unitsService struct {
	client *Client
}

var _ UnitsService = &unitsService{}

// UnitListOptions specifies options for UnitsService.List.
type UnitListOptions struct {
	// Filters
	RepositoryURI string `url:",omitempty"`
	CommitID      string `url:",omitempty"`
	UnitType      string `url:",omitempty"`
	Unit          string `url:",omitempty"`

	// Paging
	ListOptions
}

func (s *unitsService) List(opt *UnitListOptions) ([]*unit.RepoSourceUnit, Response, error) {
	url, err := s.client.url(api_router.Units, nil, opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var units []*unit.RepoSourceUnit
	resp, err := s.client.Do(req, &units)
	if err != nil {
		return nil, resp, err
	}

	return units, resp, nil
}

type MockUnitsService struct {
	List_ func(opt *UnitListOptions) ([]*unit.RepoSourceUnit, Response, error)
}

var _ UnitsService = MockUnitsService{}

func (s MockUnitsService) List(opt *UnitListOptions) ([]*unit.RepoSourceUnit, Response, error) {
	if s.List_ == nil {
		return nil, &HTTPResponse{}, nil
	}
	return s.List_(opt)
}
