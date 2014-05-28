package unit

import (
	"reflect"
	"testing"
)

type QuxSourceUnit struct {
	Bar string
}

func (u QuxSourceUnit) Name() string    { return u.Bar }
func (_ QuxSourceUnit) RootDir() string { return "qux" }
func (_ QuxSourceUnit) Paths() []string { return nil }

func TestRepoSourceUnit_SourceUnit(t *testing.T) {
	Register("Qux", &QuxSourceUnit{})
	defer unregisterSourceUnitType("Qux")

	rsu := &RepoSourceUnit{
		UnitType: "Qux",
		Data:     []byte(`{"Bar":"b"}`),
	}
	want := &QuxSourceUnit{Bar: "b"}

	u, err := rsu.SourceUnit()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(u, want) {
		t.Errorf("got %+v, want %+v", u, want)
	}
}
