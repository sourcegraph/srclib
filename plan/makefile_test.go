package plan_test

import (
	"bytes"
	"strings"
	"testing"

	"sourcegraph.com/sourcegraph/makex"
	"sourcegraph.com/sourcegraph/srclib"
	"sourcegraph.com/sourcegraph/srclib/config"
	_ "sourcegraph.com/sourcegraph/srclib/config"
	_ "sourcegraph.com/sourcegraph/srclib/dep"
	_ "sourcegraph.com/sourcegraph/srclib/grapher"
	"sourcegraph.com/sourcegraph/srclib/plan"
	"sourcegraph.com/sourcegraph/srclib/toolchain"
	"sourcegraph.com/sourcegraph/srclib/unit"
)

func TestCreateMakefile(t *testing.T) {
	oldChooseTool := toolchain.ChooseTool
	defer func() { toolchain.ChooseTool = oldChooseTool }()

	toolchain.ChooseTool = func(op, unitType string) (*srclib.ToolRef, error) {
		return &srclib.ToolRef{
			Toolchain: "tc",
			Subcmd:    "t",
		}, nil
	}
	buildDataDir := "testdata"
	c := &config.Tree{
		SourceUnits: []*unit.SourceUnit{
			{
				Key: unit.Key{
					Name: "n",
					Type: "t",
				},
				Info: unit.Info{
					Files: []string{"f"},
					Ops: map[string][]byte{
						"graph":      nil,
						"depresolve": nil,
					},
				},
			},
		},
	}

	mf, err := plan.CreateMakefile(buildDataDir, nil, "", c)
	if err != nil {
		t.Fatal(err)
	}

	want := `
.PHONY: all

all: testdata/n/t.depresolve.json testdata/n/t.graph.json

testdata/n/t.depresolve.json: testdata/n/t.unit.json
	srclib tool "tc" "t" < $^ 1> $@

testdata/n/t.graph.json: testdata/n/t.unit.json
	srclib tool "tc" "t" < $< | srclib internal normalize-graph-data --unit-type "t" --dir . 1> $@

.DELETE_ON_ERROR:
`

	gotBytes, err := makex.Marshal(mf)
	if err != nil {
		t.Fatal(err)
	}

	want = strings.TrimSpace(want)
	got := string(bytes.TrimSpace(gotBytes))

	if got != want {
		t.Errorf("got makefile:\n==========\n%s\n==========\n\nwant makefile:\n==========\n%s\n==========", got, want)
	}
}
