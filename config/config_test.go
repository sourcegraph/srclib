package config

import (
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
	Register("foo", &Foo{})
	defer unregister("foo")

	data := []byte(`{"Global": {"foo": {"bar": "qux"}}}`)
	wantConfig := &Repository{
		Global: map[string]interface{}{"foo": &Foo{Bar: "qux"}},
	}
	c, err := Read(data, "")
	if err != nil {
		t.Fatal(err)
	}

	if diff := pretty.Diff(wantConfig, c); len(diff) > 0 {
		t.Errorf("wantConfig != config\n%s", strings.Join(diff, "\n"))
	}
}
