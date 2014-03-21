package scan

import (
	"github.com/kr/pretty"
	"strings"
	"testing"
)

func TestLanguages(t *testing.T) {
	mustParseLanguages()
	if len(Languages) == 0 {
		t.Errorf("want len(Languages) > 0")
	}
}

func TestLanguagesByExtension(t *testing.T) {
	mustParseLanguages()
	wantLangs := []*Language{{
		Name:             "C",
		Type:             "programming",
		PrimaryExtension: ".c",
		Extensions:       []string{".w"},
	}}
	langs := LanguagesByExtension[".c"]
	if diff := pretty.Diff(wantLangs, langs); len(diff) > 0 {
		t.Errorf("langs\n%s", strings.Join(diff, "\n"))
	}
}
