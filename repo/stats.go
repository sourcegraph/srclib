package repo

import (
	"database/sql/driver"
	"fmt"
)

type StatType string

type Stats map[StatType]int

const (
	// StatXRefs is the number of external references to any symbol defined in a
	// repository (i.e., references from other repositories). It is only
	// computed per-repository (and not per-repository-commit) because it is
	// not easy to determine which specific commit a ref references.
	StatXRefs = "xrefs"

	// StatAuthors is the number of resolved people who contributed code to any
	// symbol defined in a repository (i.e., references from other
	// repositories). It is only computed per-repository-commit.
	StatAuthors = "authors"

	// StatClients is the number of resolved people who have committed refs that
	// reference a symbol defined in the repository. It is only computed
	// per-repository (and not per-repository-commit) because it is not easy to
	// determine which specific commit a ref references.
	StatClients = "clients"

	// StatDependencies is the number of repositories that the repository
	// depends on. It is only computed per-repository-commit.
	StatDependencies = "dependencies"

	// StatDependents is the number of repositories containing refs to a symbol
	// defined in the repository. It is only computed per-repository (and not
	// per-repository-commit) because it is not easy to determine which specific
	// commit a ref references.
	StatDependents = "dependents"

	// StatSymbols is the number of symbols defined in a repository commit. It
	// is only computed per-repository-commit (or else it would count 1 symbol
	// for each revision of the repository that we have processed).
	StatSymbols = "symbols"

	// StatExportedSymbols is the number of exported symbols defined in a
	// repository commit. It is only computed per-repository-commit (or else it
	// would count 1 symbol for each revision of the repository that we have
	// processed).
	StatExportedSymbols = "exported-symbols"
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
