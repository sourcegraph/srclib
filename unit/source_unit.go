package unit

import (
	"database/sql/driver"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

type SourceUnit struct {
	// Name is an opaque identifier for this source unit that MUST be unique
	// among all other source units of the same type in the same repository.
	//
	// Two source units of different types in a repository may have the same name.
	// To obtain an identifier for a source unit that is guaranteed to be unique
	// repository-wide, use the ID method.
	Name string

	// Type is the type of source unit this represents, such as "GoPackage".
	Type string

	// Files is all of the files that make up this source unit.
	Files []string

	// Info is an optional field that contains additional information used to
	// display the source unit
	Info *Info

	// Data is additional data dumped by the scanner about this source unit. It
	// typically holds information that the scanner wants to make available to
	// other components in the toolchain (grapher, dep resolver, etc.).
	Data interface{}
}

// idSeparator joins a source unit's name and type in its ID string.
var idSeparator = "@"

// ID returns an opaque identifier for this source unit that is guaranteed to be
// unique among all other source units in the same repository.
func (u *SourceUnit) ID() ID {
	return ID(fmt.Sprintf("%s%s%s", url.QueryEscape(u.Name), idSeparator, u.Type))
}

// ParseID parses the name and type from a source unit ID (from
// (*SourceUnit).ID()).
func ParseID(unitID string) (name, typ string, err error) {
	at := strings.Index(unitID, idSeparator)
	if at == -1 {
		return "", "", fmt.Errorf("no %q in source unit ID", idSeparator)
	}

	name, err = url.QueryUnescape(unitID[:at])
	if err != nil {
		return "", "", err
	}
	typ = unitID[at+len(idSeparator):]
	return name, typ, nil
}

// ID is a source unit ID.
type ID string

// Value implements driver.Valuer.
func (x ID) Value() (driver.Value, error) {
	return string(x), nil
}

// Scan implements sql.Scanner.
func (x *ID) Scan(v interface{}) error {
	if data, ok := v.([]byte); ok {
		*x = ID(data)
		return nil
	}
	return fmt.Errorf("%T.Scan failed: %v", x, v)
}

// ExpandPaths interprets paths, which contains paths (optionally with
// filepath.Glob-compatible globs) that are relative to base. A list of actual
// files that are referenced is returned.
func ExpandPaths(base string, paths []string) ([]string, error) {
	var expanded []string
	for _, path := range paths {
		hits, err := filepath.Glob(filepath.Join(base, path))
		if err != nil {
			return nil, err
		}
		expanded = append(expanded, hits...)
	}
	return expanded, nil
}
