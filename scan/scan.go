package scan

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"code.google.com/p/rog-go/parallel"
	"github.com/sourcegraph/srclib/config"
	"github.com/sourcegraph/srclib/repo"
	"github.com/sourcegraph/srclib/tool"
	"github.com/sourcegraph/srclib/unit"
)

// Scan returns a list of source units that exist in dir and its
// subdirectories. Paths in the source units should be relative to dir.
func Scan(dir string, tool tool.Tool) ([]*unit.SourceUnit, error) {
	cmd, err := tool.Command(dir)
	if err != nil {
		return nil, err
	}
	cmd.Args = append(cmd.Args, "scan")
	cmd.Stderr = os.Stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	var units []*unit.SourceUnit
	if err := json.NewDecoder(stdout).Decode(&units); err != nil {
		return nil, err
	}
	if err := cmd.Wait(); err != nil {
		log.Fatal(err)
	}
	return units, nil
}

var GlobalScanIgnore = []string{
	"third_party",
	"vendor",
	"bower_components",
	"node_modules",
}

// SourceUnits scans dir and its subdirectories for source units, using all
// registered toolchains that implement Scanner.
func SourceUnits(dir string, c *config.Repository) ([]*unit.SourceUnit, error) {
	tools, err := tool.SrclibPathTools.List()
	if err != nil {
		return nil, err
	}
	scanners, err := tool.FilterByHandler(tools, "scan")
	if err != nil {
		return nil, err
	}

	var units struct {
		u []*unit.SourceUnit
		sync.Mutex
	}
	run := parallel.NewRun(runtime.GOMAXPROCS(0))
	for _, s_ := range scanners {
		s := s_
		run.Do(func() error {
			log.Printf("Scanning %s using %q scanner...", c.URI, s.Name())
			units2, err := Scan(dir, s)
			if err != nil {
				log.Printf("Failed to scan %s using %q scanner: %s.", c.URI, s.Name(), err)
				return err
			}

			// Ignore source units per the config (ScanIgnore and ScanIgnoreUnitTypes).
			var units3 []*unit.SourceUnit
			for _, u := range units2 {
				ignored := false
				// for _, ignoreType := range c.ScanIgnoreUnitTypes {
				// 	if u.Type == ignoreType {
				// 		ignored = true
				// 		break
				// 	}
				// }
				// TODO(sqs): reimplement ScanIgnoreUnitTypes

				// TODO(sqs): reimplement some way of respecting c.ScanIgnore
				// and GlobalScanIgnore (SourceUnit no longer has RootDir()
				// method).
				if !ignored {
					units3 = append(units3, u)
				}
			}

			log.Printf("Finished scanning %s using %q scanner. %d source units found (after ignoring %d).", c.URI, s.Name(), len(units3), len(units2)-len(units3))

			units.Lock()
			defer units.Unlock()
			units.u = append(units.u, units3...)
			return nil
		})
	}
	if err := run.Wait(); err != nil {
		return nil, err
	}

	log.Printf("Scanning %s found %d source units total.", c.URI, len(units.u))
	return units.u, nil
}

// ReadRepositoryAndScan runs config.ReadRepository to load the repository
// configuration for the repository in dir and adds all scanned source units to
// the configuration.
func ReadRepositoryAndScan(dir string, repoURI repo.URI) (*config.Repository, error) {
	c, err := config.ReadRepository(dir, repoURI)
	if err != nil {
		return nil, err
	}

	units, err := SourceUnits(dir, c)
	if err != nil {
		return nil, err
	}

	existingUnitIDs := make(map[unit.ID]struct{}, len(units))
	for _, u := range units {
		existingUnitIDs[u.ID()] = struct{}{}
	}

	for _, u := range units {
		// Don't add this source unit if one with the same ID already exists.
		// That indicates that it was overridden and should not be automatically
		// added.
		if _, exists := existingUnitIDs[u.ID()]; !exists {
			c.SourceUnits = append(c.SourceUnits, u)
		}
	}

	return c, nil
}

// dirsContains returns true if maybeChildDir is equal to any of dirs or their
// recursive subdirectories, by purely lexical processing.
func dirsContains(dirs []string, maybeChildDir string) bool {
	for _, dir := range dirs {
		if dirContains(dir, maybeChildDir) {
			return true
		}
	}
	return false
}

// dirContains returns true if maybeChildDir is dir or one of dir's recursive
// subdirectories, by purely lexical processing.
func dirContains(dir, maybeChildDir string) bool {
	dir, maybeChildDir = filepath.Clean(dir), filepath.Clean(maybeChildDir)
	return dir == maybeChildDir || strings.HasPrefix(maybeChildDir, dir+string(filepath.Separator))
}
