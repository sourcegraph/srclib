package config

import (
	"encoding/json"
	"reflect"
	"testing"

	"sourcegraph.com/sourcegraph/srcgraph/unit"

	"github.com/kr/pretty"
)

type DummyPackage struct {
	Dir string
}

func (_ DummyPackage) ID() string      { return "dummy" }
func (_ DummyPackage) Name() string    { return "dummy" }
func (_ DummyPackage) RootDir() string { return "dummy" }
func (p DummyPackage) Paths() []string { return []string{p.Dir} }

func unregisterSourceUnitType(name string) {
	delete(unit.TypeNames, reflect.TypeOf(unit.Types[name]))
	delete(unit.Types, name)
}

func TestUnmarshal_RejectInvalidFilePaths(t *testing.T) {
	unit.Register("Dummy", &DummyPackage{})
	defer unregisterSourceUnitType("Dummy")

	tests := map[string][]byte{
		"absolute path":            []byte(`{"SourceUnits": [{"Type": "Dummy", "Dir": "/foo"}]}`),
		"relative path above root": []byte(`{"SourceUnits": [{"Type": "Dummy", "Dir": "../foo"}]}`),
	}

	for label, test := range tests {
		var config *Repository
		err := json.Unmarshal(test, &config)
		if err != nil {
			t.Fatal(err)
		}
		if err := config.validate(); err != ErrInvalidFilePath {
			t.Errorf("%s: want ErrInvalidFilePath, got err == %v", label, err)
			if config != nil {
				t.Errorf("%s: got non-nil config == %s", label, pretty.Formatter(config))
			}
		}
	}
}
