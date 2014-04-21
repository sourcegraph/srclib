package config

import "sourcegraph.com/sourcegraph/srcgraph/repo"

// repoOverrides contains config overrides for repositories that need special
// handling.
var repoOverrides = map[repo.URI]*Repository{
	"code.google.com/p/go": &Repository{
		ScanIgnore: []string{"./misc", "./test", "./doc", "./cmd"},
	},
}
