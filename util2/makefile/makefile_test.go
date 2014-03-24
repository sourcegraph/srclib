package makefile

import (
	"strings"
	"testing"
)

type file struct {
	name string
}

func (f *file) Name() string { return f.name }

type dummyRule struct {
	target  *file
	prereqs []string
	recipes []CommandRecipe
}

func (r *dummyRule) Target() Target    { return r.target }
func (r *dummyRule) Prereqs() []string { return r.prereqs }
func (r *dummyRule) Recipes() []Recipe { return toRecipes(r.recipes) }

func toRecipes(crs []CommandRecipe) []Recipe {
	rs := make([]Recipe, len(crs))
	for i, f := range crs {
		rs[i] = f
	}
	return rs
}

func TestMakefile(t *testing.T) {
	tests := []struct {
		rules    []Rule
		makefile string
	}{
		{
			rules: []Rule{
				&dummyRule{
					&file{"myTarget"},
					[]string{"myPrereq0", "myPrereq1"},
					[]CommandRecipe{{"foo", "bar"}},
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
		makefile, err := Makefile(test.rules)
		if err != nil {
			t.Error(err)
			continue
		}
		if got, want := string(makefile), strings.TrimPrefix(test.makefile, "\n"); got != want {
			t.Errorf("bad Makefile\n=========== got Makefile\n%s\n\n=========== want Makefile\n%s", got, want)
		}
	}
}
