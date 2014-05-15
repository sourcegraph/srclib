package graph

import (
	"database/sql/driver"
	"fmt"
)

type Stats map[StatType]int

type StatType string

const (
	// StatXRefs is the number of external references to a symbol (i.e.,
	// references from other repositories). It is only computed for abstract
	// symbols (see the docs for SymbolKey) because it is not easy to determine
	// which specific commit a ref references (for external refs).
	StatXRefs = "xrefs"

	// StatRRefs is the number of references to a symbol from the same
	// repository in which the symbol is defined. It is inclusive of the
	// StatURefs count. It is only computed for concrete symbols (see the docs
	// for SymbolKey) because otherwise it would count 1 rref for each unique
	// revision of the repository that we have processed. (It is easy to
	// determine which specific commit an internal ref references; we just
	// assume it references a symbol in the same commit.)
	StatRRefs = "rrefs"

	// StatURefs is the number of references to a symbol from the same source
	// unit in which the symbol is defined. It is included in the StatRRefs
	// count. It is only computed for concrete symbols (see the docs for
	// SymbolKey) because otherwise it would count 1 uref for each revision of
	// the repository that we have processed.
	StatURefs = "urefs"

	// StatAuthors is the number of distinct resolved people who contributed
	// code to a symbol's definition (according to a VCS "blame" of the
	// version). It is only computed for concrete symbols (see the docs for
	// SymbolKey).
	StatAuthors = "authors"

	// StatClients is the number of distinct resolved people who have committed
	// refs that reference a symbol. It is only computed for abstract symbols
	// (see the docs for SymbolKey) because it is not easy to determine which
	// specific commit a ref references.
	StatClients = "clients"

	// StatClients is the number of distinct repositories that contain refs that
	// reference a symbol. It is only computed for abstract symbols (see the
	// docs for SymbolKey) because it is not easy to determine which specific
	// commit a ref references.
	StatDependents = "dependents"

	// StatExportedElements is the number of exported symbols whose path is a
	// descendant of this symbol's path (and that is in the same repository and
	// source unit). It is only computed for concrete symbols (see the docs for
	// SymbolKey) because otherwise it would count 1 exported element for each
	// revision of the repository that we have processed.
	StatExportedElements = "exported-elements"

	// StatInterfaces is the number of interfaces that a symbol implements (in
	// its own repository or other repositories). TODO(sqs): it is not currently
	// being computed.
	StatInterfaces = "interfaces"

	// StatImplementations is the number of implementations of an interface
	// symbol (in its own repository or other repositories). TODO(sqs): it is
	// not currently being computed.
	StatImplementations = "implementations"
)

func (x StatType) Value() (driver.Value, error) {
	return string(x), nil
}

func (x *StatType) Scan(v interface{}) error {
	if data, ok := v.([]byte); ok {
		*x = StatType(data)
		return nil
	}
	return fmt.Errorf("%T.Scan failed: %v", x, v)
}

// UniqueRefSymbols groups refs by the RefSymbolKey field and returns a map of
// how often each RefSymbolKey appears. If m is non-nil, counts are incremented
// and a new map is not created.
func UniqueRefSymbols(refs []*Ref, m map[RefSymbolKey]int) map[RefSymbolKey]int {
	if m == nil {
		m = make(map[RefSymbolKey]int)
	}
	for _, ref := range refs {
		m[ref.RefSymbolKey()]++
	}
	return m
}
