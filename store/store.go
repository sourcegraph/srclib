package store

import (
	"errors"

	"sourcegraph.com/sourcegraph/srclib/graph"
)

// A UnitStore stores and accesses srclib build data for a single
// source unit.
type UnitStore interface {
	// Def gets a single def by its key. If no such def exists, an
	// error satisfying IsNotExist is returned.
	Def(graph.DefKey) (*graph.Def, error)

	// Defs returns all defs that match the filter.
	Defs(DefFilter) ([]*graph.Def, error)

	// Refs returns all refs that match the filter.
	Refs(RefFilter) ([]*graph.Ref, error)

	// Import imports defs, refs, etc., into the store. It overwrites
	// all existing data for this source unit (and at the commit, if
	// applicable).
	Import(graph.Output) error
}

// A DefFilter is used to filter a list of defs to only those for
// which the func returns true.
type DefFilter func(*graph.Def) bool

func allDefs(*graph.Def) bool { return true }

func defKeyFilter(key graph.DefKey) DefFilter {
	return func(def *graph.Def) bool {
		return def.DefKey == key
	}
}

// A RefFilter is used to filter a list of refs to only those for
// which the func returns true.
type RefFilter func(*graph.Ref) bool

func allRefs(*graph.Ref) bool { return true }

// IsDefNotExist returns a boolean indicating whether err is known to
// report that a def does not exist.
func IsDefNotExist(err error) bool {
	return err == errDefNotExist
}

var errDefNotExist = errors.New("def does not exist")
