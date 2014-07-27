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
	"github.com/sourcegraph/srclib/scan"
	"github.com/sourcegraph/srclib/toolchain"
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

type ConfigCmd struct {
	config.Command

	ToolchainExecOpt `group:"execution"`

	Output struct {
		Output string `short:"o" long:"output" description:"output format" default:"text" value-name:"text|json"`
	} `group:"output"`

	Args struct {
		Dir Directory `name:"DIR" default:"." description:"root directory of tree to configure"`
	} `positional-args:"yes"`
}

var configCmd ConfigCmd

func (c *ConfigCmd) Execute(args []string) error {
	if c.Args.Dir != "" {
		log.Fatalf("Currently, only configuring the current directory tree is supported (i.e., no DIR argument). You provided %q.\n\nTo configure that directory, `cd %s` in your shell and rerun this command.", c.Args.Dir, c.Args.Dir)
	}

	if c.Subdir != "." {
		// TODO(sqs): if we have overridden a repo, then we specify the
		// overridden config from the root dir of the repo. so, if you try to
		// configure from a subdir in an overridden repo, the config will be
		// wrong. disable this for now.
		log.Fatalf("Configuration is currently only supported at the root (top-level directory) of a repository, not in a subdirectory (%q).", c.Subdir)
	}

	cfg, err := config.ReadRepository(string(c.Args.Dir), repo.URI(c.Repo))
	if err != nil {
		log.Fatal(err)
	}

	if cfg.SourceUnits != nil {
		log.Fatal("Specifying source units in the Srcfile is not currently supported.")
	}

	if cfg.Scanners == nil {
		cfg.Scanners = config.SrclibPathConfig.DefaultScanners
	}

	scanners := make([]toolchain.Tool, len(cfg.Scanners))
	for i, scannerRef := range cfg.Scanners {
		scanner, err := toolchain.OpenTool(scannerRef.Toolchain, scannerRef.Subcmd, c.ToolchainMode())
		if err != nil {
			return err
		}
		scanners[i] = scanner
	}

	units, err := scan.ScanMulti(scanners, scan.Command{c.Command})
	if err != nil {
		log.Fatal(err)
	}

	// TODO(sqs): merge the Srcfile's source units with the ones we scanned;
	// don't just clobber them.
	cfg.SourceUnits = units

	currentRepo, err := OpenRepo(Dir)
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
	for _, u := range units {
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

		fmt.Printf("CONFIG (%d)\n", len(cfg.Config))
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
