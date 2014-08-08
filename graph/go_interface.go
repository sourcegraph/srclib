package graph

import "sourcegraph.com/sourcegraph/srclib/repo"

// GoInterfaceMethod represents a Go interface method defined by an Go interface
// def or implemented by a Go type def. It is used for finding all
// implementations of an interface.
type GoInterfaceMethod struct {
	// OfDefPath refers to the Go interface def that defines this method, or the Go
	// type def that implements this method.
	OfDefPath DefPath `db:"of_def_path"`

	// OfDefUnit refers to the unit containing the def denoted in OfDefPath.
	OfDefUnit string `db:"of_def_unit"`

	// Repo refers to the repository in which this method was defined.
	Repo repo.URI

	// Key is the canonical signature of the method for the implements
	// operation. If a type's methods' keys are a superset of an interface's,
	// then the type implements the interface.
	CanonicalSignature string `db:"canonical_signature"`

	// Name is the method's name.
	Name string
}
