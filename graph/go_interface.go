package graph

import (
	"sourcegraph.com/sourcegraph/srcgraph/repo"
)

// GoInterfaceMethod represents a Go interface method defined by an Go interface
// symbol or implemented by a Go type symbol. It is used for finding all
// implementations of an interface.
type GoInterfaceMethod struct {
	// Path refers to the Go interface symbol that defines this method, or the Go
	// type symbol that implements this method.
	OfSymbol SymbolPath `db:"of_symbol"`

	// OfUnit refers to the unit containing the symbol denoted in OfSymbol.
	OfUnit string `db:"of_unit"`

	// Repo refers to the repository in which this method was defined.
	Repo repo.URI 

	// Key is the canonical signature of the method for the implements
	// operation. If a type's methods' keys are a superset of an interface's,
	// then the type implements the interface.
	CanonicalSignature string `db:"canonical_signature"`

	// Name is the method's name.
	Name string 
}
