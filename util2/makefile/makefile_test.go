package makefile

import (
	"strings"
	"testing"
)

type dummyRule struct {
	target  string
	prereqs []string
	recipes []string
}

func (r *dummyRule) Target() string    { return r.target }
func (r *dummyRule) Prereqs() []string { return r.prereqs }
func (r *dummyRule) Recipes() []string { return r.recipes }

func TestMakefile(t *testing.T) {
	tests := []struct {
		rules    []Rule
		makefile string
	}{
		{
			rules: []Rule{
				&dummyRule{
					"myTarget",
					[]string{"myPrereq0", "myPrereq1"},
					[]string{"foo bar"},
				},
			},
			makefile: `
all: myTarget

myTarget: myPrereq0 myPrereq1
	foo bar
`,
		},
	}
	for _, test := range tests {
		makefile, err := Makefile(test.rules, nil)
		if err != nil {
			t.Error(err)
			continue
		}
		if got, want := string(makefile), strings.TrimPrefix(test.makefile, "\n"); got != want {
			t.Errorf("bad Makefile\n=========== got Makefile\n%s\n\n=========== want Makefile\n%s", got, want)
		}
	}
}
