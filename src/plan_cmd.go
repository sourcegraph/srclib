package src

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kr/fs"
	"github.com/sourcegraph/makex"
	"github.com/sourcegraph/rwvfs"
	"sourcegraph.com/sourcegraph/srclib/buildstore"
	"sourcegraph.com/sourcegraph/srclib/config"
	"sourcegraph.com/sourcegraph/srclib/toolchain"

	"sourcegraph.com/sourcegraph/srclib/plan"
	"sourcegraph.com/sourcegraph/srclib/unit"
)

func init() {
	c, err := CLI.AddCommand("plan",
		"generate a Makefile to process a project",
		`Generate a Makefile to process a repository or directory tree.

Requires that "src config" has already been run.
`,
		&planCmd,
	)
	if err != nil {
		log.Fatal(err)
	}

	_ = c
}

type PlanCmd struct {
	ToolchainExecOpt ToolchainExecOpt

	Args struct {
		Dir Directory `name:"DIR" default:"." description:"root directory of tree to plan"`
	} `positional-args:"yes"`
}

var planCmd PlanCmd

type walkableFileSystem struct{ rwvfs.FileSystem }

func (_ walkableFileSystem) Join(elem ...string) string { return filepath.Join(elem...) }

func (c *PlanCmd) Execute(args []string) error {
	if c.Args.Dir == "" {
		c.Args.Dir = "."
	}

	currentRepo, err := OpenRepo(string(c.Args.Dir))
	if err != nil {
		return err
	}
	buildStore, err := buildstore.NewRepositoryStore(currentRepo.RootDir)
	if err != nil {
		return err
	}
	buildDataDir, err := buildstore.BuildDir(buildStore, currentRepo.CommitID)
	if err != nil {
		return err
	}
	buildDataDir, _ = filepath.Rel(absDir, buildDataDir)

	// Get all .srclib-cache/**/*.unit.v0.json files.
	var unitFiles []string
	unitSuffix := buildstore.DataTypeSuffix(unit.SourceUnit{})
	dataPath := buildStore.CommitPath(currentRepo.CommitID)
	var w *fs.Walker
	needsCommitPrefix := false
	if fi, err := buildStore.Lstat(dataPath); os.IsNotExist(err) {
		return fmt.Errorf("build cache dir does not exist. Did you run `src config`? [dataPath=%q]", dataPath)
	} else if err != nil {
		return err
	} else if fi.Mode().IsDir() {
		w = fs.WalkFS(dataPath, buildStore)
	} else if fi.Mode()&os.ModeSymlink > 0 {
		dst, err := os.Readlink(buildDataDir)
		if err != nil {
			return err
		}
		w = fs.WalkFS(".", walkableFileSystem{rwvfs.OS(dst)})
		needsCommitPrefix = true
	} else {
		return fmt.Errorf("invalid build cache dir")
	}
	for w.Step() {
		if strings.HasSuffix(w.Path(), unitSuffix) {
			path := w.Path()
			if needsCommitPrefix {
				path = filepath.Join(currentRepo.CommitID, path)
			}
			unitFiles = append(unitFiles, path)
		}
	}

	if len(unitFiles) == 0 {
		return fmt.Errorf("no source unit files found. Did you run `src config`?")
	}

	if gopt.Verbose {
		log.Printf("Found %d source unit definition files: %v", len(unitFiles), unitFiles)
	}

	toolchainExecOptArgs, err := toolchain.MarshalArgs(&c.ToolchainExecOpt)
	if err != nil {
		return err
	}

	// Parse units
	sort.Strings(unitFiles)
	units := make([]*unit.SourceUnit, len(unitFiles))
	for i, unitFile := range unitFiles {
		f, err := buildStore.Open(unitFile)
		if err != nil {
			return err
		}
		var u *unit.SourceUnit
		if err := json.NewDecoder(f).Decode(&u); err != nil {
			f.Close()
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
		units[i] = u
	}

	mf, err := plan.CreateMakefile(buildDataDir, &config.Tree{SourceUnits: units}, plan.Options{strings.Join(toolchainExecOptArgs, " ")})
	if err != nil {
		return err
	}

	mfData, err := makex.Marshal(mf)
	if err != nil {
		log.Fatal(err)
	}
	mfFile := buildStore.FilePath(currentRepo.CommitID, "Makefile")
	if err := rwvfs.MkdirAll(buildStore, filepath.Dir(mfFile)); err != nil {
		return err
	}
	f, err := buildStore.Create(mfFile)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(mfData); err != nil {
		return err
	}

	log.Printf("Wrote %s", filepath.Join(buildDataDir, "..", mfFile))

	return nil
}
