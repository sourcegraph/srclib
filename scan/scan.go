package scan

import (
	"log"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"code.google.com/p/rog-go/parallel"
	"github.com/sourcegraph/srclib/config"
	"github.com/sourcegraph/srclib/repo"
	"github.com/sourcegraph/srclib/unit"
)

// Scanner implementations scan for source units in a repository.
type Scanner interface {
	// Scan returns a list of source units that exist in dir and its
	// subdirectories. Paths in the source units should be relative to dir.
	Scan(dir string, c *config.Repository) ([]*unit.SourceUnit, error)
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
	var units struct {
		u []*unit.SourceUnit
		sync.Mutex
	}
	run := parallel.NewRun(runtime.GOMAXPROCS(0))
	for name_, s_ := range []Scanner{} {
		name, s := name_, s_
		run.Do(func() error {
			log.Printf("Scanning %s using %q scanner...", c.URI, name)
			units2, err := s.Scan(dir, c)
			if err != nil {
				log.Printf("Failed to scan %s using %q scanner: %s.", c.URI, name, err)
				return err
			}

			// Ignore source units per the config (ScanIgnore and ScanIgnoreUnitTypes).
			var units3 []*unit.SourceUnit
			for _, u := range units2 {
				ignored := false
				for _, ignoreType := range c.ScanIgnoreUnitTypes {
					if u.Type == ignoreType {
						ignored = true
						break
					}
				}
				// TODO(sqs): reimplement some way of respecting c.ScanIgnore
				// and GlobalScanIgnore (SourceUnit no longer has RootDir()
				// method).
				if !ignored {
					units3 = append(units3, u)
				}
			}

			log.Printf("Finished scanning %s using %q scanner. %d source units found (after ignoring %d).", c.URI, name, len(units3), len(units2)-len(units3))

			units.Lock()
			defer units.Unlock()
			units.u = append(units.u, units3...)
			return nil
		})
	}
	err := run.Wait()
	log.Printf("Scanning %s found %d source units total.", c.URI, len(units.u))

	return units.u, err
}

// ReadDirConfigAndScan runs config.ReadDir to load the repository configuration
// for the repository in dir and adds all scanned source units to the
// configuration.
func ReadDirConfigAndScan(dir string, repoURI repo.URI) (*config.Repository, error) {
	c, err := config.ReadDir(dir, repoURI)
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
