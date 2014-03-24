package unit

import (
	"database/sql/driver"
	"fmt"
	"reflect"
)

var Types = make(map[string]SourceUnit)
var TypeNames = make(map[reflect.Type]string)

// Register makes a source unit available by the provided type name. The
// emptyInstance should be an empty instance of a struct (or some other type)
// that implements SourceUnit; it is used to instantiate instances dynamically.
// If Register is called twice with the same type or type name, if name is
// empty, or if emptyInstance is nil, it panics
func Register(name string, emptyInstance SourceUnit) {
	if _, dup := Types[name]; dup {
		panic("unit: Register called twice for type name " + name)
	}
	if emptyInstance == nil {
		panic("unit: Register emptyInstance is nil")
	}
	Types[name] = emptyInstance

	typ := reflect.TypeOf(emptyInstance)
	if _, dup := TypeNames[typ]; dup {
		panic("unit: Register called twice for type " + typ.String())
	}
	if name == "" {
		panic("unit: Register name is nil")
	}
	TypeNames[typ] = name
	TypeNames[reflect.PtrTo(typ)] = name
}

type SourceUnit interface {
	// Name is an identifier for this source unit that MUST be unique among all
	// other source units of the same type in the same repository.
	//
	// Two source units of different types in a repository may have the same name.
	// To obtain an identifier for a source unit that is guaranteed to be unique
	// repository-wide, use the MakeID function.
	Name() string

	// RootDir is the deepest directory that contains all files in this source
	// unit.
	RootDir() string

	// Paths returns all of the file or directory paths that this source unit
	// refers to.
	Paths() []string
}

func Type(u SourceUnit) string {
	return TypeNames[reflect.TypeOf(u)]
}

func MakeID(u SourceUnit) ID {
	return ID(fmt.Sprintf("%s@%s", u.Name(), Type(u)))
}

type ID string

func (x ID) Value() (driver.Value, error) {
	return string(x), nil
}

func (x *ID) Scan(v interface{}) error {
	if data, ok := v.([]byte); ok {
		*x = ID(data)
		return nil
	}
	return fmt.Errorf("%T.Scan failed: %v", x, v)
}
