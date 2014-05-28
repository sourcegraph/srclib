package unit

import (
	"encoding/json"
	"fmt"
	"reflect"
)

type SourceUnits []SourceUnit

// AddIfNotExists adds unit to the list of source units. If a source unit
// already exists with the same ID as u, nothing is done.
func (us *SourceUnits) AddIfNotExists(u SourceUnit) {
	unitID := MakeID(u)
	for _, u2 := range *us {
		if MakeID(u2) == unitID {
			return
		}
	}
	*us = append(*us, u)
}

// MarshalJSON implements encoding/json.Marshaler to marshal to a JSON array
// where each element is a JSON-encoded SourceUnit with an additional property,
// "Type", denoting the registered type name of the source unit.
func (us SourceUnits) MarshalJSON() ([]byte, error) {
	m := make([]map[string]interface{}, len(us))

	for i, u := range us {
		// Create a map from the struct.
		um, err := unmarshalAsUntyped(u)
		if err != nil {
			return nil, err
		}

		typ := reflect.TypeOf(u)
		if typeName, registered := TypeNames[typ]; registered {
			um["Type"] = typeName
		} else {
			return nil, fmt.Errorf("no type name for unregistered type: %s", typ)
		}

		m[i] = um
	}

	return json.Marshal(m)
}

// UnmarshalJSON implements encoding/json.Unmarshaler to unmarshal to a slice
// whose elements are struct-typed for registered source unit types.
func (u *SourceUnits) UnmarshalJSON(data []byte) error {
	var s []map[string]interface{}
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}

	// Unmarshal all registered source unit types into typed structs.
	for i, e := range s {
		typeName, _ := e["Type"].(string)
		if typeName == "" {
			return fmt.Errorf(`source unit at index %d is missing "Type"`, i)
		}
		if emptyInstance, registered := Types[typeName]; registered {
			typed := reflect.New(reflect.TypeOf(emptyInstance).Elem()).Interface()
			err = unmarshalAsTyped(e, typed)
			if err != nil {
				return err
			}
			*u = append(*u, reflect.ValueOf(typed).Interface().(SourceUnit))
		} else {
			return fmt.Errorf("unrecognized source unit type %q", typeName)
		}
	}

	return nil
}

// unmarshalAsTyped marshals orig, which should be the originally unmarshaled
// data structure (such as map[string]interface{}), and unmarshals it into
// typed, which should be a struct.
func unmarshalAsTyped(orig interface{}, typed interface{}) error {
	data, err := json.Marshal(orig)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, typed)
}

// unmarshalAsUntyped marshals orig, which is usually a struct, into JSON and
// then to a map[string]interface{}.
func unmarshalAsUntyped(orig interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(orig)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	err = json.Unmarshal(data, &m)
	if err != nil {
		return nil, err
	}
	return m, nil
}
