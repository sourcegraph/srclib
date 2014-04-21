package graph

import (
	"database/sql/driver"
	"fmt"
)

type SymbolStat struct {
	SymbolCommitKey
	Type StatType
	N    int
}

type StatType string

const (
	StatXRefs            = "xrefs"
	StatRRefs            = "rrefs"
	StatURefs            = "urefs"
	StatXRefsRecursive   = "xrefs-recursive"
	StatRRefsRecursive   = "rrefs-recursive"
	StatURefsRecursive   = "urefs-recursive"
	StatAuthors          = "authors"
	StatClients          = "clients"
	StatDependents       = "dependents"
	StatExportedElements = "exported-elements"
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
// how often each RefSymbolKey appears.
func UniqueRefSymbols(refs []*Ref) map[RefSymbolKey]int {
	m := make(map[RefSymbolKey]int)
	for _, ref := range refs {
		m[ref.RefSymbolKey()]++
	}
	return m
}
