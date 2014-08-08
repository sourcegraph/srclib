package repo

import (
	"database/sql/driver"
	"fmt"
)

// StatType is the name of a repository statistic (see below for a listing).
type StatType string

// Stats holds statistics for a repository.
type Stats map[StatType]int

const (
	// StatXRefs is the number of external references to any def defined in a
	// repository (i.e., references from other repositories). It is only
	// computed per-repository (and not per-repository-commit) because it is
	// not easy to determine which specific commit a ref references.
	StatXRefs = "xrefs"

	// StatAuthors is the number of resolved people who contributed code to any
	// def defined in a repository (i.e., references from other
	// repositories). It is only computed per-repository-commit.
	StatAuthors = "authors"

	// StatClients is the number of resolved people who have committed refs that
	// reference a def defined in the repository. It is only computed
	// per-repository (and not per-repository-commit) because it is not easy to
	// determine which specific commit a ref references.
	StatClients = "clients"

	// StatDependencies is the number of repositories that the repository
	// depends on. It is only computed per-repository-commit.
	StatDependencies = "dependencies"

	// StatDependents is the number of repositories containing refs to a def
	// defined in the repository. It is only computed per-repository (and not
	// per-repository-commit) because it is not easy to determine which specific
	// commit a ref references.
	StatDependents = "dependents"

	// StatDefs is the number of defs defined in a repository commit. It
	// is only computed per-repository-commit (or else it would count 1 def
	// for each revision of the repository that we have processed).
	StatDefs = "defs"

	// StatExportedDefs is the number of exported defs defined in a
	// repository commit. It is only computed per-repository-commit (or else it
	// would count 1 def for each revision of the repository that we have
	// processed).
	StatExportedDefs = "exported-defs"
)

var StatTypes = map[StatType]struct{}{StatXRefs: struct{}{}, StatAuthors: struct{}{}, StatClients: struct{}{}, StatDependencies: struct{}{}, StatDependents: struct{}{}, StatDefs: struct{}{}, StatExportedDefs: struct{}{}}

// Value implements database/sql/driver.Valuer.
func (x StatType) Value() (driver.Value, error) {
	return string(x), nil
}

// Scan implements database/sql.Scanner.
func (x *StatType) Scan(v interface{}) error {
	if data, ok := v.([]byte); ok {
		*x = StatType(data)
		return nil
	}
	return fmt.Errorf("%T.Scan failed: %v", x, v)
}
