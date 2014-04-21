package repo

import (
	"database/sql/driver"
	"fmt"
)

type StatType string

const (
	StatXRefs           = "xrefs"
	StatAuthors         = "authors"
	StatClients         = "clients"
	StatDependencies    = "dependencies"
	StatDependents      = "dependents"
	StatSymbols         = "symbols"
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
