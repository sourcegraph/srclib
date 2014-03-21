package config

import (
	"encoding/json"
	"reflect"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
	"strings"
	"testing"

	"github.com/kr/pretty"
)

func unregister(name string) {
	delete(Globals, name)
}

func TestRead_Global(t *testing.T) {
	type Foo struct {
		Bar string
	}
	Register("foo", Foo{})
	defer unregister("foo")

	data := []byte(`{"Global": {"foo": {"bar": "qux"}}}`)
	wantConfig := &Repository{
		Global: map[string]interface{}{"foo": Foo{Bar: "qux"}},
	}
	c, err := Read(data, "")
	if err != nil {
		t.Fatal(err)
	}

	if diff := pretty.Diff(wantConfig, c); len(diff) > 0 {
		t.Errorf("wantConfig != config\n%s", strings.Join(diff, "\n"))
	}
}

func unregisterSourceUnitType(name string) {
	delete(unit.TypeNames, reflect.TypeOf(unit.Types[name]))
	delete(unit.TypeNames, reflect.PtrTo(reflect.TypeOf(unit.Types[name])))
	delete(unit.Types, name)
}

type FooSourceUnit struct {
	Bar string
}

func (u FooSourceUnit) ID() string      { return "foo:" + u.Bar }
func (_ FooSourceUnit) Name() string    { return "foo" }
func (_ FooSourceUnit) RootDir() string { return "foo" }
func (_ FooSourceUnit) Paths() []string { return nil }

func TestSourceUnits_Unmarshal(t *testing.T) {
	unit.Register("Foo", FooSourceUnit{})
	defer unregisterSourceUnitType("Foo")

	data := []byte(`[{"Type": "Foo", "Bar": ""}, {"Type": "Foo", "Bar": "qux"}]`)
	wantUnits := SourceUnits{
		FooSourceUnit{Bar: ""},
		FooSourceUnit{Bar: "qux"},
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
	unit.Register("Foo", FooSourceUnit{})
	defer unregisterSourceUnitType("Foo")

	units := SourceUnits{
		FooSourceUnit{Bar: ""},
		FooSourceUnit{Bar: "qux"},
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
		FooSourceUnit{Bar: ""},
		FooSourceUnit{Bar: "qux"},
	}

	// Add duplicate.
	units.AddIfNotExists(FooSourceUnit{Bar: ""})
	if len(units) != 2 {
		t.Errorf("got len %d, want 2")
	}

	// Add new.
	units.AddIfNotExists(FooSourceUnit{Bar: "new"})
	if len(units) != 3 {
		t.Errorf("got len %d, want 3")
	}
}
