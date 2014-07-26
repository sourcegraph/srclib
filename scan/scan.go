package scan

import (
	"fmt"
	"log"

	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"code.google.com/p/rog-go/parallel"
	"github.com/sourcegraph/srclib/config"
	"github.com/sourcegraph/srclib/src"
	"github.com/sourcegraph/srclib/toolchain"
	"github.com/sourcegraph/srclib/unit"
	"github.com/sqs/go-flags"
)

func init() {
	var err error
	scanCmd, err = src.CLI.AddCommand("scan",
		"scan for source units",
		"Scans for source units in the directory tree rooted at the current directory.",
		&Command{},
	)
	if err != nil {
		log.Fatal(err)
	}

	src.SetRepoOptDefaults(scanCmd)

	_, err = scanCmd.AddGroup("execution options (not passed to tools)", "", &execOpt)
	if err != nil {
		log.Fatal(err)
	}

	// Set default scanners.
	cfg, err := config.ReadRepository(".", "")
	if err != nil {
		log.Fatal(err)
	}
	configGroup, err := scanCmd.AddGroup("configuration (not passed to tools)", "", &configOpt)
	if err != nil {
		log.Fatal(err)
	}
	defaultScanners := make([]string, len(cfg.Scanners))
	for i, sref := range cfg.Scanners {
		sstr, err := sref.MarshalFlag()
		if err != nil {
			log.Fatal(err)
		}
		defaultScanners[i] = sstr
	}
	src.SetOptionDefaultValue(configGroup, "tool", defaultScanners...)
}

var (
	scanCmd   *flags.Command
	execOpt   src.ToolchainExecOpt
	configOpt struct {
		Scanners []toolchain.ToolRef `short:"t" long:"tool" description:"(list) scanner tools to run" value-name:"TOOLREF"`
	}
)

type Command struct {
	Repo   string `long:"repo" description:"repository URI" value-name:"URI"`
	Subdir string `long:"subdir" description:"subdirectory in repository" value-name:"DIR"`
}

func (c *Command) Execute(args []string) error {
	var units struct {
		u []*unit.SourceUnit
		sync.Mutex
	}
	run := parallel.NewRun(runtime.GOMAXPROCS(0))
	for _, tool_ := range configOpt.Scanners {
		// TODO(sqs): how to specify whether to use program or docker tool here?
		tool := tool_
		run.Do(func() error {
			log.Printf("Scanning %s using %q scanner...", c.Repo, tool)
			units2, err := Scan(tool, c)
			if err != nil {
				return err
			}

			units.Lock()
			defer units.Unlock()
			units.u = append(units.u, units2...)
			return nil
		})
	}
	if err := run.Wait(); err != nil {
		return err
	}

	log.Printf("Scanning %s: found %d source units total.", c.Repo, len(units.u))

	for _, u := range units.u {
		fmt.Println(u.ID())
	}

	return nil
}

func Scan(tool toolchain.ToolRef, cmd *Command) ([]*unit.SourceUnit, error) {
	s, err := toolchain.OpenTool(tool.Toolchain, tool.Subcmd, execOpt.ToolchainMode())
	if err != nil {
		return nil, err
	}

	args, err := src.MarshalArgs(scanCmd.Group)
	if err != nil {
		return nil, err
	}

	var units []*unit.SourceUnit
	if err := s.Run(args, &units); err != nil {
		log.Printf("Failed to scan using %q scanner: %s.", tool, err)
		return nil, err
	}

	for _, u := range units {
		u.Scanner = tool
	}

	log.Printf("Finished scanning using %q scanner: %d source units found.", tool, len(units))
	return units, nil
}

////////////////////////////////////////////////////////////////////////////////
// TODO(sqs): everything below here is legacy/unused
////////////////////////////////////////////////////////////////////////////////

// ReadRepositoryAndScan runs config.ReadRepository to load the repository
// configuration for the repository in dir and adds all scanned source units to
// the configuration.
// func ReadRepositoryAndScan(dir string, repoURI repo.URI) (*config.Repository, error) {
// 	c, err := config.ReadRepository(dir, repoURI)
// 	if err != nil {
// 		return nil, err
// 	}

// 	units, err := SourceUnits(dir, c)
// 	if err != nil {
// 		return nil, err
// 	}

// 	existingUnitIDs := make(map[unit.ID]struct{}, len(units))
// 	for _, u := range c.SourceUnits {
// 		existingUnitIDs[u.ID()] = struct{}{}
// 	}

// 	for _, u := range units {
// 		// Don't add this source unit if one with the same ID already exists.
// 		// That indicates that it was overridden and should not be automatically
// 		// added.
// 		if _, exists := existingUnitIDs[u.ID()]; !exists {
// 			c.SourceUnits = append(c.SourceUnits, u)
// 		}
// 	}

// 	return c, nil
// }

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

var GlobalScanIgnore = []string{
	"third_party",
	"vendor",
	"bower_components",
	"node_modules",
}
