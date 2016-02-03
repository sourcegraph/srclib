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
	"strings"

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

type CoverageCmd struct {
	w io.Writer
}

type Coverage struct {
	Repo     *Repo
	Warnings []string
}

var coverageCmd CoverageCmd

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
	for _, p := range strings.Split(goPath, ":") {
		if strings.Contains(lRepo.RootDir, p) {
			importPath = strings.TrimPrefix(lRepo.RootDir, filepath.Join(p, "src"))
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
			// should we output the current coverage anyway? (aka just break)
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
			// this should work, so it 	qualifies as an error case
			return err
		}

		_, pkgErr := build.ImportDir(pth, 0)

		// TODO(poler) allow the user to specify an older commit (fine for now)
		cachePath := filepath.Join(lRepo.RootDir, cacheDir, lRepo.CommitID, importPath, relPath)

		_, unitErr := os.Stat(filepath.Join(cachePath, unitFile))

		if unitErr == nil && pkgErr != nil {
			cov.Warnings = append(cov.Warnings, fmt.Sprintf("GoPackage.unit.json existed but importing the package at %s failed", pth))
		} else if unitErr != nil && pkgErr == nil {
			cov.Warnings = append(cov.Warnings, fmt.Sprintf("GoPackage.unit.json did not exist but importing the package at %s succeeded", pth))

		}

	}

	enc := json.NewEncoder(c.w)
	if err := enc.Encode(cov); err != nil {
		return err
	}

	return nil
}
