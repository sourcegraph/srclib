package plan_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/sourcegraph/makex"
	_ "sourcegraph.com/sourcegraph/srclib/authorship"
	"sourcegraph.com/sourcegraph/srclib/config"
	_ "sourcegraph.com/sourcegraph/srclib/config"
	_ "sourcegraph.com/sourcegraph/srclib/dep"
	_ "sourcegraph.com/sourcegraph/srclib/grapher"
	"sourcegraph.com/sourcegraph/srclib/plan"
	"sourcegraph.com/sourcegraph/srclib/toolchain"
	"sourcegraph.com/sourcegraph/srclib/unit"
	_ "sourcegraph.com/sourcegraph/srclib/vcsutil"
)

func TestCreateMakefile(t *testing.T) {
	buildDataDir := "testdata"
	c := &config.Tree{
		SourceUnits: []*unit.SourceUnit{
			{
				Name:  "n",
				Type:  "t",
				Files: []string{"f"},
				Ops: map[string]*toolchain.ToolRef{
					"graph":      {Toolchain: "tc", Subcmd: "t"},
					"depresolve": {Toolchain: "tc", Subcmd: "t"},
				},
			},
		},
	}

	mf, err := plan.CreateMakefile(buildDataDir, c, plan.Options{})
	if err != nil {
		t.Fatal(err)
	}

	want := `
all: testdata/n/t.blame.json testdata/n/t.graph.json testdata/n/t.depresolve.json testdata/n/t.authorship.json

testdata/n/t.blame.json: testdata/n/t.unit.json f
	src internal unit-blame --unit-data testdata/n/t.unit.json 1> $@

testdata/n/t.graph.json: testdata/n/t.unit.json f
	src tool  "tc" "t" < $^ | src internal normalize-graph-data 1> $@

testdata/n/t.depresolve.json: testdata/n/t.unit.json
	src tool  "tc" "t" < $^ 1> $@

testdata/n/t.authorship.json: testdata/n/t.blame.json testdata/n/t.graph.json
	src internal unit-authorship --blame-data testdata/n/t.blame.json --graph-data testdata/n/t.graph.json 1> $@

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
