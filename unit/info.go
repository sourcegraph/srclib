package unit

import (
	"sourcegraph.com/sourcegraph/srcgraph/repo"
)

// Info is implemented by source units that want to provide additional
// information for display purposes. If a SourceUnit doesn't implement Info,
// GetInfo will return an Info with default return values for each method.
type Info interface {
	// NameInRepository is the name to use when displaying the source unit in
	// the context of the repository in which it is defined. The defining
	// repository's URI is provided as the defining repo.URI argument. This name
	// typically needs less qualification than GlobalName.
	//
	// For example, a Go package's GlobalName is its repository URI basename
	// plus its directory path within the repository (e.g.,
	// "github.com/user/repo/x/y"'s NameInRepository is "repo/x/y"). Because npm
	// and pip packages are named globally, their name is probably appropriate
	// to use as both the unit's NameInRepository and GlobalName.
	NameInRepository(repo.URI) string

	// GlobalName is the name to use when displaying the source unit *OUTSIDE OF*
	// the context of the repository in which it is defined.
	//
	// For example, a Go package's GlobalName is its full import path. Because
	// npm and pip packages are named globally, their name is probably
	// appropriate to use as both the unit's NameInRepository and GlobalName.
	GlobalName() string

	// Description is a short (~1-sentence) description of the source unit.
	Description() string

	// Type is the human-readable name of the type of source unit; e.g., "Go
	// package".
	Type() string
}

func GetInfo(u SourceUnit) Info {
	if u, ok := u.(Info); ok {
		return u
	}
	return defaultInfo{u}
}

// defaultInfo is a default implementation of Info for source units that don't
// implement Info themselves.
type defaultInfo struct{ SourceUnit }

// NameInRepository implements Info.
func (d defaultInfo) NameInRepository(_ repo.URI) string { return d.SourceUnit.Name() }

// GlobalName implements Info.
func (d defaultInfo) GlobalName() string { return d.SourceUnit.Name() }

// Description implements Info.
func (d defaultInfo) Description() string { return "" }

// Type implements Info.
func (d defaultInfo) Type() string { return Type(d.SourceUnit) }
