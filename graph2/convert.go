package graph2

import (
	"encoding/json"
	"fmt"

	"sourcegraph.com/sourcegraph/srclib/unit"
)

// This file contains functions to convert from srclib 1.0 datastructures to 2.0

func NewUnit(u *unit.SourceUnit) (*Unit, error) {
	var key UnitKey
	key.TreeType = "git"
	key.URI = u.Repo
	key.Version = u.CommitID
	key.UnitName = u.Name
	key.UnitType = u.Type

	dataBytes, err := json.Marshal(u.Data)
	if err != nil {
		return nil, err
	}

	newDeps, err := convertDeps(u.Dependencies)
	if err != nil {
		return nil, err
	}

	return &Unit{
		UnitKey: key,
		Globs:   u.Globs,
		Files:   u.Files,
		Dir:     u.Dir,
		Deps:    newDeps,
		Info: &UnitInfo{
			NameInRepository: u.Info.NameInRepository,
			GlobalName:       u.Info.GlobalName,
			Description:      u.Info.Description,
			TypeName:         u.Info.TypeName,
		},
		Data: dataBytes,
	}, nil
}

func convertDeps(oldDeps []interface{}) (newDeps []*Dep, err error) {
	for _, dep := range oldDeps {
		// TODO
		switch dep.(type) {
		default:
			return nil, fmt.Errorf("unsupported dep type: %T: %+v", dep, dep)
		}
	}
	return newDeps, nil
}
