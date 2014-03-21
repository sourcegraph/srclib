package scan

import (
	"code.google.com/p/rog-go/parallel"
	"runtime"

	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/repo"
	"sourcegraph.com/sourcegraph/srcgraph/task2"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
	"sync"
)

// Scanner implementations scan for source units in a repository.
type Scanner interface {
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
		c.SourceUnits.AddIfNotExists(u)
	}

	return c, nil
}
