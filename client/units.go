package client

import (
	"sourcegraph.com/sourcegraph/api_router"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
)

// UnitsService communicates with the source unit-related endpoints in
// the Sourcegraph API.
type UnitsService interface {
	// Get fetches a unit.
	Get(spec *UnitSpec) (*unit.RepoSourceUnit, Response, error)

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

func SpecFromUnit(u *unit.RepoSourceUnit) *UnitSpec {
	return &UnitSpec{
		Repo:     string(u.Repo),
		CommitID: u.CommitID,
		UnitType: u.UnitType,
		Unit:     u.Unit,
	}
}

func UnitSpecFromRouteVars(vars map[string]string) *UnitSpec {
	return &UnitSpec{
		Repo:     vars["RepoURI"],
		CommitID: vars["Rev"],
		UnitType: vars["UnitType"],
		Unit:     vars["Unit"],
	}
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

func (s *unitsService) Get(spec *UnitSpec) (*unit.RepoSourceUnit, Response, error) {
	url, err := s.client.url(api_router.Unit, spec.RouteVars(), nil)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var u unit.RepoSourceUnit
	resp, err := s.client.Do(req, &u)
	if err != nil {
		return nil, resp, err
	}

	return &u, resp, nil
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
	Get_  func(spec *UnitSpec) (*unit.RepoSourceUnit, Response, error)
}

var _ UnitsService = MockUnitsService{}

func (s MockUnitsService) List(opt *UnitListOptions) ([]*unit.RepoSourceUnit, Response, error) {
	if s.List_ == nil {
		return nil, &HTTPResponse{}, nil
	}
	return s.List_(opt)
}

func (s MockUnitsService) Get(spec *UnitSpec) (*unit.RepoSourceUnit, Response, error) {
	if s.Get_ == nil {
		return nil, &HTTPResponse{}, nil
	}
	return s.Get_(spec)
}
