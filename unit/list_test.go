package unit

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/kr/pretty"
)

func unregisterSourceUnitType(name string) {
	delete(TypeNames, reflect.TypeOf(Types[name]))
	delete(Types, name)
}

type FooSourceUnit struct {
	Bar string
}

func (u FooSourceUnit) Name() string    { return u.Bar }
func (_ FooSourceUnit) RootDir() string { return "foo" }
func (_ FooSourceUnit) Paths() []string { return nil }

func TestSourceUnits_Unmarshal(t *testing.T) {
	Register("Foo", &FooSourceUnit{})
	defer unregisterSourceUnitType("Foo")

	data := []byte(`[{"Type": "Foo", "Bar": ""}, {"Type": "Foo", "Bar": "qux"}]`)
	wantUnits := SourceUnits{
		&FooSourceUnit{Bar: ""},
		&FooSourceUnit{Bar: "qux"},
	}
	var u SourceUnits
	err := json.Unmarshal(data, &u)
	if err != nil {
		t.Fatal(err)
	}

	if diff := pretty.Diff(wantUnits, u); len(diff) > 0 {
		t.Errorf("wantUnits != units\n%s", strings.Join(diff, "\n"))
	}
}

func TestSourceUnits_Marshal(t *testing.T) {
	Register("Foo", &FooSourceUnit{})
	defer unregisterSourceUnitType("Foo")

	units := SourceUnits{
		&FooSourceUnit{Bar: ""},
		&FooSourceUnit{Bar: "qux"},
	}
	wantData := []byte(`[{"Bar":"","Type":"Foo"},{"Bar":"qux","Type":"Foo"}]`)
	data, err := json.Marshal(units)
	if err != nil {
		t.Fatal(err)
	}

	if diff := pretty.Diff(string(wantData), string(data)); len(diff) > 0 {
		t.Errorf("wantData != data\n%s", strings.Join(diff, "\n"))
	}
}

func TestSourceUnits_AddIfNotExists(t *testing.T) {
	units := SourceUnits{
		&FooSourceUnit{Bar: ""},
		&FooSourceUnit{Bar: "qux"},
	}

	// Add duplicate.
	units.AddIfNotExists(&FooSourceUnit{Bar: ""})
	if len(units) != 2 {
		t.Errorf("got len %d, want 2", len(units))
	}

	// Add new.
	units.AddIfNotExists(&FooSourceUnit{Bar: "new"})
	if len(units) != 3 {
		t.Errorf("got len %d, want 3", len(units))
	}
}
