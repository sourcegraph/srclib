package src

import (
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"sort"

	"github.com/sourcegraph/rwvfs"
	"github.com/sourcegraph/srclib/buildstore"
	"github.com/sourcegraph/srclib/config"
	"github.com/sourcegraph/srclib/plan"
	"github.com/sourcegraph/srclib/repo"

	"github.com/sourcegraph/srclib/unit"
)

func init() {
	c, err := CLI.AddCommand("config",
		"reads & scans for project configuration",
		`Produces a configuration file suitable for building the repository or directory tree rooted at DIR (or the current directory if not specified).

The steps are:

1. Read user srclib config (SRCLIBPATH/.srclibconfig), if present.

2. Read configuration from the current directory's Srcfile (if present).

3. Scan for source units in the directory tree rooted at the current directory (or the root of the repository containing the current directory), using the scanners specified in either the user srclib config or the Srcfile (or otherwise the defaults).

The default values for --repo and --subdir are determined by detecting the current repository and reading its Srcfile config (if any).
`,
		&configCmd,
	)
	if err != nil {
		log.Fatal(err)
	}
	c.Aliases = []string{"c"}

	SetRepoOptDefaults(c)
}

// getInitialConfig gets the initial config (i.e., the config that comes solely
// from the Srcfile, if any, and the external user config, before running the
// scanners).
func getInitialConfig(opt config.Options, dir Directory) (*config.Repository, error) {
	if dir != "" && dir != "." {
		log.Fatalf("Currently, only configuring the current directory tree is supported (i.e., no DIR argument). You provided %q.\n\nTo configure that directory, `cd %s` in your shell and rerun this command.", dir, dir)
	}

	if opt.Subdir != "." {
		// TODO(sqs): if we have overridden a repo, then we specify the
		// overridden config from the root dir of the repo. so, if you try to
		// configure from a subdir in an overridden repo, the config will be
		// wrong. disable this for now.
		log.Fatalf("Configuration is currently only supported at the root (top-level directory) of a repository, not in a subdirectory (%q).", opt.Subdir)
	}

	cfg, err := config.ReadRepository(string(dir), repo.URI(opt.Repo))
	if err != nil {
		return nil, err
	}

	if cfg.SourceUnits != nil {
		// TODO(sqs): support specifying source units in the Srcfile
		log.Fatal("specifying source units in the Srcfile is not currently supported.")
	}

	if cfg.Scanners == nil {
		cfg.Scanners = config.SrclibPathConfig.DefaultScanners
	}

	return cfg, nil
}

type ConfigCmd struct {
	config.Options

	ToolchainExecOpt `group:"execution"`
	BuildCacheOpt    `group:"build cache"`

	Output struct {
		Output string `short:"o" long:"output" description:"output format" default:"text" value-name:"text|json"`
	} `group:"output"`

	Args struct {
		Dir Directory `name:"DIR" default:"." description:"root directory of tree to configure"`
	} `positional-args:"yes"`
}

var configCmd ConfigCmd

func (c *ConfigCmd) Execute(args []string) error {
	if c.Args.Dir == "" {
		c.Args.Dir = "."
	}

	cfg, err := getInitialConfig(c.Options, c.Args.Dir)
	if err != nil {
		return err
	}

	if err := scanUnitsIntoConfig(cfg, c.Options, c.ToolchainExecOpt); err != nil {
		return err
	}

	currentRepo, err := OpenRepo(string(c.Args.Dir))
	if err != nil {
		return err
	}
	buildStore, err := buildstore.NewRepositoryStore(currentRepo.RootDir)
	if err != nil {
		return err
	}

	// Write source units to build cache.
	//
	// TODO(sqs): create Makefile.config that makes it more standard to recreate
	// these when the source unit defns change (and to determine when the source
	// unit defns change), with targets like:
	//
	// UNITNAME/UNITTYPE.unit.v0.json: setup.py mylib/foo.py
	//   src config --unit=UNITNAME@UNITTYPE
	//
	// or maybe a custom stale checker is better than just using file mtimes for
	// all the files (maybe just use setup.py as a prereq? but then how will we
	// update SourceUnit.Files list? SourceUnit.Globs could help here...)
	if !c.NoCacheWrite {
		for _, u := range cfg.SourceUnits {
			filename := buildStore.FilePath(currentRepo.CommitID, plan.SourceUnitDataFilename(unit.SourceUnit{}, u))
			if err := rwvfs.MkdirAll(buildStore, filepath.Dir(filename)); err != nil {
				return err
			}
			f, err := buildStore.Create(filename)
			if err != nil {
				return err
			}
			if err := json.NewEncoder(f).Encode(u); err != nil {
				return err
			}
			if err := f.Close(); err != nil {
				log.Fatal(err)
			}
		}
	}

	if c.Output.Output == "json" {
		PrintJSON(cfg, "")
	} else {
		fmt.Printf("SCANNERS (%d)\n", len(cfg.Scanners))
		for _, s := range cfg.Scanners {
			fmt.Printf(" - %s\n", s)
		}
		fmt.Println()

		fmt.Printf("SOURCE UNITS (%d)\n", len(cfg.SourceUnits))
		for _, u := range cfg.SourceUnits {
			fmt.Printf(" - %s: %s\n", u.Type, u.Name)
		}
		fmt.Println()

		fmt.Printf("CONFIG PROPERTIES (%d)\n", len(cfg.Config))
		for _, kv := range sortedMap(cfg.Config) {
			fmt.Printf(" - %s: %q\n", kv[0], kv[1])
		}
	}

	return nil
}

func sortedMap(m map[string]string) [][2]string {
	keys := make([]string, len(m))
	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	sort.Strings(keys)

	sorted := make([][2]string, len(keys))
	for i, k := range keys {
		sorted[i] = [2]string{k, m[k]}
	}
	return sorted
}
