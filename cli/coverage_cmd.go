package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"go/build"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/tools/go/loader"

	"github.com/kr/fs"

	"sourcegraph.com/sourcegraph/go-flags"
)

func init() {
	cliInit = append(cliInit, func(cli *flags.Command) {
		_, err := cli.AddCommand("coverage",
			"use a simple heuristic to check that srclib is outputting expected graph data",
			`The coverage command acts as a sanity check to ensure that the srclib-go toolchain succeeds when it should and fails when it should not.`,
			&coverageCmd,
		)
		if err != nil {
			log.Fatal(err)
		}
	})
}

var cacheDir = ".srclib-cache"
var gitDir = ".git"
var unitFile = "GoPackage.unit.json"
var depFile = "GoPackage.depresolve.json"
var graphFile = "GoPackage.graph.json"

type CoverageCmd struct {
	w io.Writer
}

type Coverage struct {
	Repo     *Repo
	Warnings []BuildWarning
}

type BuildWarning struct {
	Directory string
	Warning   WarningType
}

type WarningType string

const (
	BuildSucceededSrclibFailed = "Build succeeded but Srclib outputs failed"
	BuildFailedSrclibSucceeded = "Build failed but Srclib succeeded"
)

var coverageCmd CoverageCmd

// Execute performs a sanity check on srclib builds for go repositories.
// We iterate every directory and do a coverage check on every package.
// The coverage heuristic is very rough currently, but makes the following assumptions:
// - only golang coverage is checked
// - if a directory can be imported (see build standard library package) and built
//   (see loader standard library package), then there should be three files present
//   in the corresponding directory under .srclib-cache: a unit file, a depresolve file,
//   and a graph file.
func (c *CoverageCmd) Execute(args []string) error {
	if c.w == nil {
		c.w = os.Stdout
	}

	lRepo, lRepoErr := OpenLocalRepo()
	if lRepoErr != nil {
		return lRepoErr
	}

	goPath := os.Getenv("GOPATH")
	if goPath == "" {
		return errors.New("GOPATH not set")
	}

	var importPath string
	var splitGoPath []string
	if runtime.GOOS == "windows" {
		splitGoPath = strings.Split(goPath, ";")
	} else {
		splitGoPath = strings.Split(goPath, ":")
	}

	for _, p := range splitGoPath {
		if strings.Contains(lRepo.RootDir, p) {
			importPath = strings.TrimPrefix(lRepo.RootDir, filepath.Join(p, "src")+"/")
		}
	}

	if importPath == "" {
		return fmt.Errorf("Unable to find an import path for the current repo at %s", lRepo.RootDir)

	}

	cov := &Coverage{
		Repo: lRepo,
	}

	_, err := os.Stat(filepath.Join(lRepo.RootDir, cacheDir))
	if os.IsNotExist(err) {
		return err
	}

	walker := fs.Walk(lRepo.RootDir)

	for walker.Step() {

		if err := walker.Err(); err != nil {
			return err
		}

		pth := walker.Path()

		fi, err := os.Stat(pth)
		if !fi.IsDir() {
			continue
		}

		if strings.Contains(pth, cacheDir) || strings.Contains(pth, gitDir) {
			continue
		}

		relPath, err := filepath.Rel(lRepo.RootDir, pth)
		if err != nil {
			return err
		}

		_, importErr := build.ImportDir(pth, 0)

		var conf loader.Config
		conf.Import(strings.Join([]string{importPath, relPath}, "/"))
		_, pkgErr := conf.Load()

		importAndBuildSucceeded := (importErr == nil) && (pkgErr == nil)

		// TODO(poler) allow the user to specify an older commit (fine for now)
		cachePath := filepath.Join(lRepo.RootDir, cacheDir, lRepo.CommitID, importPath, relPath)

		// If the srclib build config was at all customized, the assumption that these files
		// will exist is almost certainly not valid.
		unitPath := filepath.Join(cachePath, unitFile)
		depPath := filepath.Join(cachePath, depFile)
		graphPath := filepath.Join(cachePath, graphFile)

		_, unitErr := os.Stat(unitPath)
		_, depErr := os.Stat(depPath)
		_, graphErr := os.Stat(graphPath)

		srclibOutputsExist := (unitErr == nil) && (depErr == nil) && (graphErr == nil)

		if importAndBuildSucceeded && !srclibOutputsExist {
			cov.Warnings = append(cov.Warnings, BuildWarning{
				Directory: pth,
				Warning:   BuildSucceededSrclibFailed,
			})
		} else if !importAndBuildSucceeded && srclibOutputsExist {
			cov.Warnings = append(cov.Warnings, BuildWarning{
				Directory: pth,
				Warning:   BuildFailedSrclibSucceeded,
			})
		}
	}

	enc := json.NewEncoder(c.w)
	if err := enc.Encode(cov); err != nil {
		return err
	}

	return nil

}
