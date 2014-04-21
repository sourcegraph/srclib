package scan

import (
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"code.google.com/p/rog-go/parallel"
	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/repo"
	"sourcegraph.com/sourcegraph/srcgraph/task2"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
)

// Scanner implementations scan for source units in a repository.
type Scanner interface {
	// Scan returns a list of source units that exist in dir and its
	// subdirectories. Paths in the source units should be relative to dir.
	Scan(dir string, c *config.Repository, x *task2.Context) ([]unit.SourceUnit, error)
}

// Scanners holds all registered scanners.
var Scanners = make(map[string]Scanner)

// Register adds a scanner to the list of scanners used to detect source units.
// If Register is called twice with the same name or if scanner is nil, it
// panics
func Register(name string, scanner Scanner) {
	if _, dup := Scanners[name]; dup {
		panic("scan: Register called twice for name " + name)
	}
	if scanner == nil {
		panic("scan: Register scanner is nil")
	}
	Scanners[name] = scanner
}

// SourceUnits scans dir and its subdirectories for source units, using all
// registered toolchains that implement Scanner.
func SourceUnits(dir string, c *config.Repository, x *task2.Context) ([]unit.SourceUnit, error) {
	var units struct {
		u []unit.SourceUnit
		sync.Mutex
	}
	run := parallel.NewRun(runtime.GOMAXPROCS(0))
	for name, s_ := range Scanners {
		s := s_
		run.Do(func() error {
			x.Log.Printf("Scanning %s using %q scanner...", c.URI, name)
			units2, err := s.Scan(dir, c, x)
			if err != nil {
				x.Log.Printf("Failed to scan %s using %q scanner: %s.", c.URI, name, err)
				return err
			}
			x.Log.Printf("Finished scanning %s using %q scanner. Found %d source units.", c.URI, name, len(units2))

			units.Lock()
			defer units.Unlock()
			units.u = append(units.u, units2...)
			return nil
		})
	}
	err := run.Wait()
	x.Log.Printf("Scanning %s found %d source units total.", c.URI, len(units.u))

	return units.u, err
}

// ReadDirConfigAndScan runs config.ReadDir to load the repository configuration
// for the repository in dir and adds all scanned source units to the
// configuration.
func ReadDirConfigAndScan(dir string, repoURI repo.URI, x *task2.Context) (*config.Repository, error) {
	c, err := config.ReadDir(dir, repoURI)
	if err != nil {
		return nil, err
	}

	units, err := SourceUnits(dir, c, x)
	if err != nil {
		return nil, err
	}
	for _, u := range units {
		if dirsContains(c.ScanIgnore, u.RootDir()) {
			continue
		}
		c.SourceUnits.AddIfNotExists(u)
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
