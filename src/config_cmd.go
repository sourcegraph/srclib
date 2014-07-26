package src

import (
	"log"

	"github.com/sourcegraph/srclib/config"
	"github.com/sourcegraph/srclib/repo"
	"github.com/sourcegraph/srclib/scan"
	"github.com/sourcegraph/srclib/toolchain"
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
		Dir string `name:"DIR" default:"." description:"root directory of tree to configure"`
	} `positional-args:"yes"`
}

var configCmd ConfigCmd

func (c *ConfigCmd) Execute(args []string) error {
	if c.Args.Dir == "" {
		c.Args.Dir = "."
	}

	if c.Subdir != "." {
		// TODO(sqs): if we have overridden a repo, then we specify the
		// overridden config from the root dir of the repo. so, if you try to
		// configure from a subdir in an overridden repo, the config will be
		// wrong. disable this for now.
		log.Fatalf("Configuration is currently only supported at the root (top-level directory) of a repository, not in a subdirectory (%q).", c.Subdir)
	}

	cfg, err := config.ReadRepository(c.Args.Dir, repo.URI(c.Repo))
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

	cfg.SourceUnits = units

	// TODO(sqs): make this do stuff
	PrintJSON(cfg, "")

	return nil
}
