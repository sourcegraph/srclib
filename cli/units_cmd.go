package cli

import (
	"log"
	"path/filepath"

	"strings"

	"github.com/alexsaveliev/go-colorable-wrapper"
	"sourcegraph.com/sourcegraph/go-flags"

	"sourcegraph.com/sourcegraph/srclib/config"
	"sourcegraph.com/sourcegraph/srclib/scan"
	"sourcegraph.com/sourcegraph/srclib/toolchain"
	"sourcegraph.com/sourcegraph/srclib/unit"
)

func init() {
	cliInit = append(cliInit, func(cli *flags.Command) {
		_, err := cli.AddCommand("units",
			"lists source units",
			`Lists source units in the repository or directory tree rooted at DIR (or the current directory if DIR is not specified).`,
			&unitsCmd,
		)
		if err != nil {
			log.Fatal(err)
		}
	})
}

// scanUnitsIntoConfig uses cfg to scan for source units. It modifies
// cfg.SourceUnits, merging the scanned source units with those already present
// in cfg.
func scanUnitsIntoConfig(cfg *config.Repository, quiet bool) error {
	scanners := make([][]string, len(cfg.Scanners))
	for i, scannerRef := range cfg.Scanners {
		cmdName, err := toolchain.Command(scannerRef.Toolchain)
		if err != nil {
			return err
		}
		scanners[i] = []string{cmdName, scannerRef.Subcmd}
	}

	units, err := scan.ScanMulti(scanners, scan.Options{Quiet: quiet}, cfg.Config)
	if err != nil {
		return err
	}

	// Merge the repo/tree config with each source unit's config.
	if cfg.Config == nil {
		cfg.Config = map[string]interface{}{}
	}
	for _, u := range units {
		for k, v := range cfg.Config {
			if uv, present := u.Config[k]; present {
				log.Printf("Both the scanned source unit %q and the Srcfile specify a Config key %q. Using the value from the scanned source unit (%+v).", u.ID(), k, uv)
			} else {
				if u.Config == nil {
					u.Config = make(map[string]string)
				}
				if vstr, isStr := v.(string); isStr {
					u.Config[k] = vstr
				}
			}
		}
	}

	// collect manually specified source units by ID
	manualUnits := make(map[unit.ID]*unit.SourceUnit, len(cfg.SourceUnits))
	for _, u := range cfg.SourceUnits {
		manualUnits[u.ID()] = u

		xf, err := unit.ExpandPaths(".", u.Files)
		if err != nil {
			return err
		}
		u.Files = xf
	}

	for _, u := range units {
		if mu, present := manualUnits[u.ID()]; present {
			log.Printf("Found manually specified source unit %q with same ID as scanned source unit. Using manually specified unit, ignoring scanned source unit.", mu.ID())
			continue
		}

		unitDir := u.Dir
		if unitDir == "" && len(u.Files) > 0 {
			// in case the unit doesn't specify a Dir, obtain it from the first file
			unitDir = filepath.Dir(u.Files[0])
		}

		// heed SkipDirs
		if pathHasAnyPrefix(unitDir, cfg.SkipDirs) {
			continue
		}

		skip := false
		for _, skipUnit := range cfg.SkipUnits {
			if u.Name == skipUnit.Name && u.Type == skipUnit.Type {
				skip = true
				break
			}
		}
		if skip {
			continue
		}

		cfg.SourceUnits = append(cfg.SourceUnits, u)
	}

	return nil
}

type UnitsCmd struct {
	Output struct {
		Output string `short:"o" long:"output" description:"output format" default:"text" value-name:"text|json"`
	} `group:"output"`

	Args struct {
		Dir Directory `name:"DIR" default:"." description:"root directory of tree to list units in"`
	} `positional-args:"yes"`
}

var unitsCmd UnitsCmd

func (c *UnitsCmd) Execute(args []string) error {
	cfg, err := getInitialConfig(c.Args.Dir.String())
	if err != nil {
		return err
	}

	if err := scanUnitsIntoConfig(cfg, false); err != nil {
		return err
	}

	if c.Output.Output == "json" {
		PrintJSON(cfg.SourceUnits, "")
	} else {
		for _, u := range cfg.SourceUnits {
			colorable.Printf("%-50s  %s\n", u.Name, u.Type)
		}
	}

	return nil
}

func pathHasAnyPrefix(path string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if pathHasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func pathHasPrefix(path, prefix string) bool {
	path = filepath.Clean(path)
	prefix = filepath.Clean(prefix)
	return prefix == "." || path == prefix || strings.HasPrefix(path, prefix+string(filepath.Separator))
}
