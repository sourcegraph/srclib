// Package overrides defines overridden configurations for repositories that
// need special handling. It should be imported for side effects by all main
// packages that perform analysis work.
//
// It is separate from package config because it needs to import packages that
// config may not depend on (or else there will be an import cycle).
package overrides

import (
	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/repo"
	"sourcegraph.com/sourcegraph/srcgraph/toolchain/javascript"
	"sourcegraph.com/sourcegraph/srcgraph/toolchain/python"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
)

func init() {
	o := map[repo.URI]*config.Repository{
		"github.com/emicklei/go-restful": &config.Repository{
			ScanIgnore: []string{"./examples"},
		},
		"code.google.com/p/go": &config.Repository{
			ScanIgnore: []string{"./misc", "./test", "./doc", "./cmd", "./src/cmd"},
		},
		"github.com/joyent/node": &config.Repository{
			SourceUnits: unit.SourceUnits{
				&javascript.CommonJSPackage{
					Package:            []byte(`{}`),
					PackageName:        javascript.NodeJSStdlibUnit,
					PackageDescription: "The Node.js core API.",
					Dir:                ".",
					LibFiles: []string{
						"lib/assert.js",
						"lib/buffer.js",
						"lib/child_process.js",
						"lib/cluster.js",
						"lib/console.js",
						"lib/constants.js",
						"lib/crypto.js",
						"lib/dgram.js",
						"lib/dns.js",
						"lib/domain.js",
						"lib/events.js",
						"lib/freelist.js",
						"lib/fs.js",
						"lib/http.js",
						"lib/https.js",
						"lib/module.js",
						"lib/net.js",
						"lib/os.js",
						"lib/path.js",
						"lib/punycode.js",
						"lib/querystring.js",
						"lib/readline.js",
						"lib/repl.js",
						"lib/smalloc.js",
						"lib/stream.js",
						"lib/string_decoder.js",
						"lib/sys.js",
						"lib/timers.js",
						"lib/tls.js",
						"lib/tty.js",
						"lib/url.js",
						"lib/util.js",
						"lib/vm.js",
						"lib/zlib.js",
					},
				},
			},

			// Suppress the Python source unit that exists because the node
			// repo has *.py files.
			ScanIgnoreUnitTypes: []string{unit.Type(&python.DistPackage{})},
			ScanIgnore:          []string{"./tools", "./deps", "./test", "./src"},

			Global: config.Global{
				"jsg": &javascript.JSGConfig{
					Plugins: map[string]interface{}{
						// In this repository, the node core modules are in the
						// lib/ dir.
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
	}
	for repoURI, c := range o {
		config.AddOverride(repoURI, c)
	}
}
