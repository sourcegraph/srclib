package cli

import (
	"log"
	"path/filepath"

	"sourcegraph.com/sourcegraph/srclib/config"
	"sourcegraph.com/sourcegraph/srclib/graph2"
	"sourcegraph.com/sourcegraph/srclib/scan"
	"sourcegraph.com/sourcegraph/srclib/toolchain"
	"sourcegraph.com/sourcegraph/srclib/unit"
)

// scanUnitsIntoConfig2 uses cfg to scan for build units. It modifies
// cfg.Units, merging the scanned build units with those already present
// in cfg.
func scanUnitsIntoConfig2(cfg *config.Tree2, quiet bool) error {
	scanners := make([][]string, len(cfg.Scanners))
	for i, scannerRef := range cfg.Scanners {
		cmdName, err := toolchain.Command(scannerRef.Toolchain)
		if err != nil {
			return err
		}
		scanners[i] = []string{cmdName, scannerRef.Subcmd}
	}

	units, err := scan.ScanMulti2(scanners, scan.Options{Quiet: quiet}, cfg.Config)
	if err != nil {
		return err
	}

	// collect manually specified build units by ID
	manualUnits := make(map[string]*graph2.Unit, len(cfg.Units))
	for _, u := range cfg.Units {
		manualUnits[u.ID()] = u

		xf, err := unit.ExpandPaths(".", u.Files)
		if err != nil {
			return err
		}
		u.Files = xf
	}

	for _, u := range units {
		if mu, present := manualUnits[u.ID()]; present {
			log.Printf("Found manually specified build unit %q with same ID as scanned build unit. Using manually specified unit, ignoring scanned build unit.", mu.ID())
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
			if u.UnitName == skipUnit.Name && u.UnitType == skipUnit.Type {
				skip = true
				break
			}
		}
		if skip {
			continue
		}

		cfg.Units = append(cfg.Units, u)
	}

	return nil
}
