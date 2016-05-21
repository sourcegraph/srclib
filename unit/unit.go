package unit

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"sourcegraph.com/sourcegraph/srclib/buildstore"
)

func init() {
	buildstore.RegisterDataType("unit", SourceUnit{})
}

// UnitRepoUnresolved is a sentinel value that indicates this unit's
// repository is unresolved
const UnitRepoUnresolved = "?"

func (u Key) IsResolved() bool {
	return u.Repo == UnitRepoUnresolved
}

// ContainsAny returns true if u contains any files in filesnames. Currently
// doesn't process globs.
func (u SourceUnit) ContainsAny(filenames []string) bool {
	if len(filenames) == 0 {
		return false
	}
	files := make(map[string]bool)
	for _, f := range filenames {
		files[f] = true
	}
	for _, uf := range u.Files {
		if files[uf] {
			return true
		}
	}
	return false
}

// idSeparator joins a source unit's name and type in its ID string.
const idSeparator = "@"

// ID returns an opaque identifier for this source unit that is guaranteed to be
// unique among all other source units in the same repository.
func (u SourceUnit) ID() ID {
	return ID(fmt.Sprintf("%s%s%s", url.QueryEscape(u.Name), idSeparator, u.Type))
}

func (u *SourceUnit) ID2() ID2 {
	return u.Key.ID2()
}

func (u Key) ID2() ID2 {
	return ID2{Type: u.Type, Name: u.Name}
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

// ID is a source unit ID. It is only unique within a repository.
type ID string

// ID2 is a source unit ID. It is only unique within a repository.
type ID2 struct {
	Type string
	Name string
}

func (v ID2) String() string { return fmt.Sprintf("{%s %s}", v.Type, v.Name) }

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
		for _, hit := range hits {
			expanded = append(expanded, filepath.ToSlash(hit))
		}
	}
	return expanded, nil
}

type SourceUnits []*SourceUnit

func (v SourceUnits) Len() int           { return len(v) }
func (v SourceUnits) Less(i, j int) bool { return v[i].String() < v[j].String() }
func (v SourceUnits) Swap(i, j int)      { v[i], v[j] = v[j], v[i] }
