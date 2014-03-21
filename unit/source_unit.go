package unit

import (
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
	ID() string
	Name() string
	RootDir() string

	// Paths returns all of the file or directory paths that this source unit
	// refers to.
	Paths() []string
}

func SourceUnitType(u SourceUnit) string {
	return reflect.TypeOf(u).Elem().Name()
}
