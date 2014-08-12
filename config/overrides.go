package config

import (
	"sourcegraph.com/sourcegraph/srclib/repo"
	"sourcegraph.com/sourcegraph/srclib/unit"
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
	"github.com/joyent/node": {
		URI: "github.com/joyent/node",
		Tree: Tree{
			SkipDirs: []string{"tools", "deps", "test", "src"},
			SourceUnits: []*unit.SourceUnit{
				{
					Name:  "node",
					Type:  "CommonJSPackage",
					Dir:   ".",
					Files: []string{"lib/*.js"},
					Config: map[string]interface{}{
						"jsg": map[string]interface{}{
							"plugins": map[string]interface{}{
								"node": map[string]string{"coreModulesDir": "lib/"},
								"$(JSG_DIR)/node_modules/tern-node-api-doc/node-api-doc": map[string]string{
									"apiDocDir":      "doc/api/",
									"apiSrcDir":      "lib/",
									"generateJSPath": "tools/doc/generate.js",
								},
							},
						},
					},
				},
			},
		},
	},
}
