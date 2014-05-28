package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"

	"sourcegraph.com/sourcegraph/srcgraph/repo"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
)

// Globals maps registered global configuration section names to an empty struct
// describing their structure.
var Globals = make(map[string]interface{})

// Register adds a global configuration section with the given name that
// deserializes into emptyInstance. If Register is called twice with the same
// name or if emptyInstance is nil, it panics
func Register(name string, emptyInstance interface{}) {
	if _, dup := Globals[name]; dup {
		panic("unit: Register called twice for name " + name)
	}
	if emptyInstance == nil {
		panic("unit: Register emptyInstance is nil")
	}
	Globals[name] = emptyInstance
}

var Filename = ".sourcegraph"

var (
	ErrDirMismatch     = errors.New("config base dir doesn't match the dir used when marshaling")
	ErrInvalidFilePath = errors.New("invalid file path specified in config (above config root dir or source unit dir)")
)

type Repository struct {
	URI         repo.URI         `json:",omitempty"`
	SourceUnits unit.SourceUnits `json:",omitempty"`
	ScanIgnore  []string         `json:",omitempty"`

	// ScanIgnoreUnitTypes is a list of source unit type names (e.g.,
	// "GoPackage") that should be ignored if found by the scanner.
	ScanIgnoreUnitTypes []string `json:",omitempty"`

	Global Global `json:",omitempty"`
}

type Global map[string]interface{}

// ReadDir parses and validates the configuration for a repository. If no
// .sourcegraph file exists, it returns the default configuration for the
// repository.
func ReadDir(dir string, repoURI repo.URI) (*Repository, error) {
	var c *Repository
	if oc, overridden := repoOverrides[repoURI]; overridden {
		c = oc
	} else if f, err := os.Open(filepath.Join(dir, Filename)); err == nil {
		defer f.Close()
		err = json.NewDecoder(f).Decode(&c)
		if err != nil {
			return nil, err
		}
	} else if os.IsNotExist(err) {
		err = nil
		c = new(Repository)
	} else {
		return nil, err
	}

	return c.finish(repoURI)
}

// Read parses and validates the JSON representation of a repository's
// configuration. If the JSON representation is empty (len(configJSON) == 0), it
// returns the default configuration for the repository.
func Read(configJSON []byte, repoURI repo.URI) (*Repository, error) {
	var c *Repository
	if len(configJSON) > 0 {
		err := json.Unmarshal(configJSON, &c)
		if err != nil {
			return nil, err
		}
	} else {
		c = new(Repository)
	}

	return c.finish(repoURI)
}

func (c *Repository) finish(repoURI repo.URI) (*Repository, error) {
	err := c.validate()
	if err != nil {
		return nil, err
	}
	c.URI = repoURI
	return c, nil
}

// UnmarshalJSON implements encoding/json.Unmarshaler to unmarshal to a map
// whose values are struct-typed for registered global section names.
func (g *Global) UnmarshalJSON(data []byte) error {
	var m map[string]interface{}
	err := json.Unmarshal(data, &m)
	if err != nil {
		return err
	}

	// Unmarshal all registered global config sections into typed structs.
	for name, v := range m {
		if emptyInstance, registered := Globals[name]; registered {
			typed := reflect.New(reflect.TypeOf(emptyInstance).Elem()).Interface()
			err = unmarshalAsTyped(v, typed)
			if err != nil {
				return err
			}
			m[name] = reflect.ValueOf(typed).Interface()
		}
	}

	*g = m
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
