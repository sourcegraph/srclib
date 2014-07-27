package src

import (
	"fmt"
	"log"

	"github.com/sourcegraph/srclib/config"
	"github.com/sourcegraph/srclib/scan"
	"github.com/sourcegraph/srclib/toolchain"
)

func init() {
	c, err := CLI.AddCommand("units",
		"lists source units",
		`Lists source units in the repository or directory tree rooted at DIR (or the current directory if DIR is not specified).`,
		&unitsCmd,
	)
	if err != nil {
		log.Fatal(err)
	}

	SetRepoOptDefaults(c)
}

// scanIntoConfig uses cfg to scan for source units. It modifies
// cfg.SourceUnits, merging the scanned source units with those already present
// in cfg.
func scanIntoConfig(cfg *config.Repository, configOpt config.Options, execOpt ToolchainExecOpt) error {
	scanners := make([]toolchain.Tool, len(cfg.Scanners))
	for i, scannerRef := range cfg.Scanners {
		scanner, err := toolchain.OpenTool(scannerRef.Toolchain, scannerRef.Subcmd, execOpt.ToolchainMode())
		if err != nil {
			return err
		}
		scanners[i] = scanner
	}

	units, err := scan.ScanMulti(scanners, scan.Options{configOpt})
	if err != nil {
		return err
	}

	// TODO(sqs): merge the Srcfile's source units with the ones we scanned;
	// don't just clobber them.
	cfg.SourceUnits = units

	return nil
}

type UnitsCmd struct {
	config.Options

	ToolchainExecOpt `group:"execution"`

	Output struct {
		Output string `short:"o" long:"output" description:"output format" default:"text" value-name:"text|json"`
	} `group:"output"`

	Args struct {
		Dir Directory `name:"DIR" default:"." description:"root directory of tree to list units in"`
	} `positional-args:"yes"`
}

var unitsCmd UnitsCmd

func (c *UnitsCmd) Execute(args []string) error {
	if c.Args.Dir == "" {
		c.Args.Dir = "."
	}

	cfg, err := getInitialConfig(c.Options, c.Args.Dir)
	if err != nil {
		return err
	}

	if err := scanIntoConfig(cfg, c.Options, c.ToolchainExecOpt); err != nil {
		return err
	}

	if c.Output.Output == "json" {
		PrintJSON(cfg.SourceUnits, "")
	} else {
		for _, u := range cfg.SourceUnits {
			fmt.Printf("%-50s  %s\n", u.Name, u.Type)
		}
	}

	return nil
}
