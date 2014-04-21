package graph

import (
	"strconv"

	"sourcegraph.com/sourcegraph/repo"
)

type RefSymbolKey struct {
	SymbolRepo     repo.URI   `db:"symbol_repo" json:",omitempty"`
	SymbolUnitType string     `db:"symbol_unit_type" json:",omitempty"`
	SymbolUnit     string     `db:"symbol_unit" json:",omitempty"`
	SymbolPath     SymbolPath `db:"symbol_path" json:",omitempty"`
}

type RefKey struct {
	SymbolRepo     repo.URI   `db:"symbol_repo" json:",omitempty"`
	SymbolUnitType string     `db:"symbol_unit_type" json:",omitempty"`
	SymbolUnit     string     `db:"symbol_unit" json:",omitempty"`
	SymbolPath     SymbolPath `db:"symbol_path" json:",omitempty"`
	Def            bool       `json:",omitempty"`
	Repo           repo.URI   `json:",omitempty"`
	UnitType       string     `db:"unit_type" json:",omitempty"`
	Unit           string     `json:",omitempty"`
	File           string     `json:",omitempty"`
	CommitID       string     `db:"commit_id" json:",omitempty"`
	Start          int        `json:",omitempty"`
	End            int        `json:",omitempty"`
}

// Ref represents a reference from source code to a symbol.
type Ref struct {
	SymbolRepo     repo.URI   `db:"symbol_repo"`
	SymbolUnitType string     `db:"symbol_unit_type"`
	SymbolUnit     string     `db:"symbol_unit"`
	SymbolPath     SymbolPath `db:"symbol_path"`

	// Def is true if this ref is to a definition of the target symbol.
	Def bool

	Repo repo.URI `json:"repo"`

	// CommitID is the immutable commit ID (not the branch name) of the VCS
	// revision that this ref was found in.
	CommitID string `db:"commit_id" json:",omitempty"`

	UnitType string `db:"unit_type" json:",omitempty"`
	Unit     string `json:",omitempty"`

	File  string `json:"file"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}

func (r *Ref) RefKey() RefKey {
	return RefKey{
		SymbolRepo:     r.SymbolRepo,
		SymbolUnitType: r.SymbolUnitType,
		SymbolUnit:     r.SymbolUnit,
		SymbolPath:     r.SymbolPath,
		Def:            r.Def,
		Repo:           r.Repo,
		UnitType:       r.UnitType,
		Unit:           r.Unit,
		File:           r.File,
		Start:          r.Start,
		End:            r.End,
	}
}

func (r *Ref) RefSymbolKey() RefSymbolKey {
	return RefSymbolKey{
		SymbolRepo:     r.SymbolRepo,
		SymbolUnitType: r.SymbolUnitType,
		SymbolUnit:     r.SymbolUnit,
		SymbolPath:     r.SymbolPath,
	}
}

func (r *Ref) SymbolKey() SymbolKey {
	return SymbolKey{
		Repo:     r.SymbolRepo,
		UnitType: r.SymbolUnitType,
		Unit:     r.SymbolUnit,
		Path:     r.SymbolPath,
	}
}

func (r *Ref) SetFromSymbolKey(k SymbolKey) {
	r.SymbolPath = k.Path
	r.SymbolUnitType = k.UnitType
	r.SymbolUnit = k.Unit
	r.SymbolRepo = k.Repo
}

// Sorting

type Refs []*Ref

func (r *Ref) sortKey() string {
	return string(r.SymbolPath) + string(r.SymbolRepo) + r.SymbolUnitType + r.SymbolUnit + string(r.Repo) + r.UnitType + r.Unit + r.File + strconv.Itoa(r.Start) + strconv.Itoa(r.End)
}
func (vs Refs) Len() int           { return len(vs) }
func (vs Refs) Swap(i, j int)      { vs[i], vs[j] = vs[j], vs[i] }
func (vs Refs) Less(i, j int) bool { return vs[i].sortKey() < vs[j].sortKey() }

// RefSet is a set of Refs. It can used to determine whether a grapher emits
// duplicate refs.
type RefSet struct {
	refs map[Ref]struct{}
}

func NewRefSet() *RefSet {
	return &RefSet{make(map[Ref]struct{})}
}

// AddAndCheckUnique adds ref to the set of seen refs, and returns whether the
// ref already existed in the set.
func (c *RefSet) AddAndCheckUnique(ref Ref) (duplicate bool) {
	_, present := c.refs[ref]
	if present {
		return true
	}
	c.refs[ref] = struct{}{}
	return false
}
