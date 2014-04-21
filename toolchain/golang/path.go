package golang

import (
	"strings"

	"sourcegraph.com/sourcegraph/srcgraph/graph"
	"sourcegraph.com/sourcegraph/srcgraph/repo"
)

// parseSymbolPath parses a Go symbol path such as "github.com/foo/bar/Baz" into
// 2 parts: repo ("github.com/foo/bar") and symbol ("github.com/foo/bar/Baz")
// paths. The symbol path includes the repo because Go considers symbols to be
// defined by their full import path.
func parseSymbolPath(path graph.SymbolPath) (r repo.URI, sym graph.SymbolPath) {
	sym = path
	parts := strings.Split(string(path), "/")

	// grapher test repos
	parts[0] = strings.ToLower(parts[0])
	if len(parts) >= 2 && (parts[0] == "case" || parts[0] == "launchpad.net" || parts[0] == "gist.github.com") {
		r = repo.URI(strings.Join(parts[:2], "/"))
		return
	}
	if len(parts) < 3 {
		panic("parseSymbolPath: bad path: '" + string(path) + "'")
	}

	r = repo.URI(strings.Join(parts[:3], "/"))
	return
}
