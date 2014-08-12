package config

import (
	"sourcegraph.com/sourcegraph/srclib/repo"
)

var overrides = map[repo.URI]*Repository{
	"code.google.com/p/go": {
		URI: "code.google.com/p/go",
		Tree: Tree{
			Config:            map[string]interface{}{"GOROOT": "."},
			SkipDirs:          []string{"test", "misc", "doc", "lib", "include"},
			PreConfigCommands: []string{"cd src && ./make.bash"},
		},
	},
}
