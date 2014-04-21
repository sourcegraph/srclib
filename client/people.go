package client

import (
	"sourcegraph.com/sourcegraph/api_router"
	"sourcegraph.com/sourcegraph/srcgraph/person"
)

type PeopleService interface {
	Get(person PersonSpec) (*person.User, *Response, error)
	List(opt *PersonListOptions) ([]*person.User, *Response, error)
	ListAuthors(person PersonSpec, opt *PersonListOptions) ([]*AugmentedPersonRef, *Response, error)
	ListClients(person PersonSpec, opt *PersonListOptions) ([]*AugmentedPersonRef, *Response, error)
}

type peopleService struct {
	client *Client
}

var _ PeopleService = &peopleService{}

type PersonSpec struct {
	LoginOrEmail string
}

func (s *peopleService) Get(person_ PersonSpec) (*person.User, *Response, error) {
	url, err := s.client.url(api_router.Person, map[string]string{"PersonSpec": person_.LoginOrEmail}, nil)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var person__ *person.User
	resp, err := s.client.Do(req, &person__)
	if err != nil {
		return nil, resp, err
	}

	return person__, resp, nil
}

type PersonListOptions struct {
	Query string `url:",omitempty"`

	Sort      string `url:",omitempty"`
	Direction string `url:",omitempty"`

	ListOptions
}

func (s *peopleService) List(opt *PersonListOptions) ([]*person.User, *Response, error) {
	url, err := s.client.url(api_router.People, nil, opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var people []*person.User
	resp, err := s.client.Do(req, &people)
	if err != nil {
		return nil, resp, err
	}

	return people, resp, nil
}

// AugmentedPersonRef is a rel.PersonRef with the full person.User struct embedded.
type AugmentedPersonRef struct {
	User  *person.User `json:"user"`
	Count int          `json:"count"`
}

func (s *peopleService) listPersonPersonRefs(person PersonSpec, routeName string, opt interface{}) ([]*AugmentedPersonRef, *Response, error) {
	url, err := s.client.url(routeName, map[string]string{"PersonSpec": person.LoginOrEmail}, opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var people []*AugmentedPersonRef
	resp, err := s.client.Do(req, &people)
	if err != nil {
		return nil, resp, err
	}

	return people, resp, nil
}

func (s *peopleService) ListAuthors(person PersonSpec, opt *PersonListOptions) ([]*AugmentedPersonRef, *Response, error) {
	return s.listPersonPersonRefs(person, api_router.PersonAuthors, opt)
}

func (s *peopleService) ListClients(person PersonSpec, opt *PersonListOptions) ([]*AugmentedPersonRef, *Response, error) {
	return s.listPersonPersonRefs(person, api_router.PersonClients, opt)
}

type MockPeopleService struct {
	Get_         func(person PersonSpec) (*person.User, *Response, error)
	List_        func(opt *PersonListOptions) ([]*person.User, *Response, error)
	ListAuthors_ func(person PersonSpec, opt *PersonListOptions) ([]*AugmentedPersonRef, *Response, error)
	ListClients_ func(person PersonSpec, opt *PersonListOptions) ([]*AugmentedPersonRef, *Response, error)
}

var _ PeopleService = MockPeopleService{}

func (s MockPeopleService) Get(person PersonSpec) (*person.User, *Response, error) {
	if s.Get_ == nil {
		return nil, &Response{}, nil
	}
	return s.Get_(person)
}

func (s MockPeopleService) List(opt *PersonListOptions) ([]*person.User, *Response, error) {
	if s.List_ == nil {
		return nil, &Response{}, nil
	}
	return s.List_(opt)
}

func (s MockPeopleService) ListAuthors(person PersonSpec, opt *PersonListOptions) ([]*AugmentedPersonRef, *Response, error) {
	if s.ListAuthors_ == nil {
		return nil, &Response{}, nil
	}
	return s.ListAuthors_(person, opt)
}

func (s MockPeopleService) ListClients(person PersonSpec, opt *PersonListOptions) ([]*AugmentedPersonRef, *Response, error) {
	if s.ListClients_ == nil {
		return nil, &Response{}, nil
	}
	return s.ListClients_(person, opt)
}
