package javascript

import (
	"encoding/json"

	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/container"
	"sourcegraph.com/sourcegraph/srcgraph/grapher2"
	"sourcegraph.com/sourcegraph/srcgraph/task2"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
)

func init() {
	unit.Register("CommonJSPackage", &CommonJSPackage{})
	grapher2.Register(&CommonJSPackage{}, grapher2.DockerGrapher{defaultJSG})
}

type CommonJSPackage struct {
	// If the field names of CommonJSPackage change, you need to EITHER (1)
	// update commonjs-findpkgs or (2) add a Transform func in the scanner to
	// map from the commonjs-findpkgs output to []*CommonJSPackage.

	// Dir is the directory that immediately contains the package.json
	// file (or would if one existed).
	Dir string

	// PackageJSONFile is the path to the package.json file, or empty if none
	// exists.
	PackageJSONFile string

	LibFiles  []string
	TestFiles []string
}

func (p CommonJSPackage) Name() string    { return p.Dir }
func (p CommonJSPackage) RootDir() string { return p.Dir }
func (p CommonJSPackage) sourceFiles() []string {
	return append(append([]string{}, p.LibFiles...), p.TestFiles...)
}
func (p CommonJSPackage) Paths() []string {
	f := p.sourceFiles()
	if p.PackageJSONFile != "" {
		f = append(f, p.PackageJSONFile)
	}
	return f
}

type jsg struct{ nodeVersion }

var defaultJSG = &jsg{defaultNode}

func (v jsg) BuildGrapher(dir string, u unit.SourceUnit, c *config.Repository, x *task2.Context) (*container.Command, error) {
	pkg := u.(*CommonJSPackage)

	if len(pkg.sourceFiles()) == 0 {
		// No source files found for source unit; proceed without running grapher.
		return nil, nil
	}

	dockerfile, err := v.baseDockerfile()
	if err != nil {
		return nil, err
	}

	// Install VCS tools in Docker container.
	const (
		jsgVersion = "jsg@0.0.1"
		jsgGit     = "git://github.com/sourcegraph/jsg.git"
		jsgSrc     = jsgGit
	)
	dockerfile = append(dockerfile, []byte("\n\nRUN npm install -g "+jsgSrc+"\n")...)

	jsgPlugins := map[string]interface{}{}
	isStdlib := false
	if isStdlib {
		// Use the lib/ dir in the node repository itself.
	} else {
		// Use the node_core_modules dir that ships with jsg (for resolving refs to the node core).

		// Copy node_core_modules to a separate dir so that refs to them aren't interpreted as refs to jsg.
		nodeCoreModulesDir := "/tmp/node_core_modules"
		dockerfile = append(dockerfile, []byte("\nRUN cp -R /usr/local/lib/node_modules/jsg/testdata/node_core_modules "+nodeCoreModulesDir+"\n")...)

		jsgPlugins["node"] = struct {
			CoreModulesDir string `json:"coreModulesDir"`
		}{nodeCoreModulesDir}
	}

	jsgCmd, err := jsgCommand(jsgPlugins, nil, nil, pkg.sourceFiles())
	if err != nil {
		return nil, err
	}

	// Track test files so we can set the Test field on symbols efficiently.
	isTestFile := make(map[string]struct{}, len(pkg.TestFiles))
	for _, f := range pkg.TestFiles {
		isTestFile[f] = struct{}{}
	}

	containerDir := containerDir(dir)
	cmd := container.Command{
		Container: container.Container{
			Dockerfile:       dockerfile,
			AddDirs:          [][2]string{{dir, containerDir}},
			PreCmdDockerfile: []byte("WORKDIR " + containerDir + "\nRUN npm install --ignore-scripts --no-bin-links"),
			Cmd:              jsgCmd,
			Dir:              containerDir,
			Stderr:           x.Stderr,
			Stdout:           x.Stdout,
		},
		Transform: func(in []byte) ([]byte, error) {
			var o jsgOutput
			err := json.Unmarshal(in, &o)
			if err != nil {
				return nil, err
			}

			var o2 grapher2.Output

			for _, js := range o.Symbols {
				sym, refs, propgs, docs, err := convertSymbol(js)
				if err != nil {
					return nil, err
				}

				if _, isTest := isTestFile[sym.File]; isTest {
					sym.Test = true
				}

				o2.Symbols = append(o2.Symbols, sym)
				o2.Refs = append(o2.Refs, refs...)
				// TODO(sqs): handle propgs
				_ = propgs
				o2.Docs = append(o2.Docs, docs...)
			}
			for _, jr := range o.Refs {
				ref, err := convertRef(u, jr)
				if err != nil {
					return nil, err
				}
				if ref != nil {
					o2.Refs = append(o2.Refs, ref)
				}
			}

			return json.Marshal(o2)
		},
	}

	return &cmd, nil
}
