package config

import (
	"strings"

	"sourcegraph.com/sourcegraph/srcgraph/repo"
)

// AddOverride overrides the configuration for the repository. It must be called
// by a package that config does not import, because the package that calls
// AddOverride typically needs to import toolchain packages, which package
// config can't depend on (or there will be an import cycle).
func AddOverride(repoURI repo.URI, config *Repository) {
	repoURI = repo.URI(strings.ToLower(string(repoURI)))
	if _, exists := repoOverrides[repoURI]; exists {
		panic("AddOverride: repository " + string(repoURI) + " already has overridden config")
	}
	if config == nil {
		panic("AddOverride: config == nil")
	}
	repoOverrides[repoURI] = config
}

// repoOverride returns the overridden configuration for the repository, or nil
// if it hasn't been overridden.
func repoOverride(repoURI repo.URI) *Repository {
	return repoOverrides[repo.URI(strings.ToLower(string(repoURI)))]
}

// repoOverrides contains config overrides for repositories that need special
// handling.
var repoOverrides = make(map[repo.URI]*Repository)
