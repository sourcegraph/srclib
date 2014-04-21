package golang

import (
	"testing"

	"sourcegraph.com/sourcegraph/repo"
	"sourcegraph.com/sourcegraph/srcgraph/graph"
)

func TestParseSymbolPath(t *testing.T) {
	tests := []struct {
		path graph.SymbolPath
		repo repo.URI
		sym  graph.SymbolPath
	}{
		{"github.com/user/repo", "github.com/user/repo", "github.com/user/repo"},
		{"github.com/user/repo/sym", "github.com/user/repo", "github.com/user/repo/sym"},
		{"github.com/user/repo/sym", "github.com/user/repo", "github.com/user/repo/sym"},
		{"gist.github.com/1234.git/sym", "gist.github.com/1234.git", "gist.github.com/1234.git/sym"},
		{"code.google.com/p/repo/sym", "code.google.com/p/repo", "code.google.com/p/repo/sym"},
		{"code.google.com/p/repo/sym", "code.google.com/p/repo", "code.google.com/p/repo/sym"},
		{"code.google.com/p/go/src/pkg/net/http", "code.google.com/p/go", "code.google.com/p/go/src/pkg/net/http"},
		{"launchpad.net/repo", "launchpad.net/repo", "launchpad.net/repo"},
		{"launchpad.net/repo/sym", "launchpad.net/repo", "launchpad.net/repo/sym"},

		// for grapher tests
		{"case/foo/bar", "case/foo", "case/foo/bar"},
	}
	for _, test := range tests {
		repo, sym := parseSymbolPath(test.path)
		if test.repo != repo {
			t.Errorf("%s: want repo %q, got %q", test.path, test.repo, repo)
		}
		if test.sym != sym {
			t.Errorf("%s: want sym %q, got %q", test.path, test.sym, sym)
		}
	}
}
