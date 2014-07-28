package src

import (
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/kr/fs"
	"github.com/sourcegraph/makex"
	"github.com/sourcegraph/rwvfs"
	"github.com/sourcegraph/srclib/buildstore"
	"github.com/sourcegraph/srclib/toolchain"

	"github.com/sourcegraph/srclib/plan"
	"github.com/sourcegraph/srclib/unit"
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
	Args struct {
		Dir Directory `name:"DIR" default:"." description:"root directory of tree to plan"`
	} `positional-args:"yes"`
}

var planCmd PlanCmd

func (c *PlanCmd) Execute(args []string) error {
	if c.Args.Dir == "" {
		c.Args.Dir = "."
	}

	// Get all .srclib-cache/**/*.unit.v0.json files.
	currentRepo, err := OpenRepo(string(c.Args.Dir))
	if err != nil {
		return err
	}
	buildStore, err := buildstore.NewRepositoryStore(currentRepo.RootDir)
	if err != nil {
		return err
	}
	var unitFiles []string
	unitSuffix := buildstore.DataTypeSuffix(unit.SourceUnit{})
	w := fs.WalkFS(buildStore.CommitPath(currentRepo.CommitID), buildStore)
	for w.Step() {
		if strings.HasSuffix(w.Path(), unitSuffix) {
			unitFiles = append(unitFiles, w.Path())
		}
	}

	if len(unitFiles) == 0 {
		return fmt.Errorf("no source unit files found. Did you run `src config`?")
	}

	buildDataDir, err := buildstore.BuildDir(buildStore, currentRepo.CommitID)
	if err != nil {
		return err
	}
	buildDataDir, _ = filepath.Rel(absDir, buildDataDir)

	var mf makex.Makefile
	var allTargets []string
	for _, unitFile := range unitFiles {
		f, err := buildStore.Open(unitFile)
		if err != nil {
			return err
		}
		var u *unit.SourceUnit
		if err := json.NewDecoder(f).Decode(&u); err != nil {
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}

		// TODO(sqs): make the "graph" target depend on the "depresolve" target
		// to avoid duplicating work
		for op, toolRef := range u.Ops {
			// TODO(sqs): actually discover which tools to use
			if toolRef == nil {
				switch op {
				case "graph":
					toolRef = &toolchain.ToolRef{Toolchain: "github.com/sourcegraph/srclib-go", Subcmd: "graph"}
				case "depresolve":
					toolRef = &toolchain.ToolRef{Toolchain: "github.com/sourcegraph/srclib-go", Subcmd: "depresolve"}
				default:
					return fmt.Errorf("no tool found for op %q on unit type %q", op, u.Type)
				}
			}
			target := filepath.Join(buildDataDir, plan.SourceUnitDataFilename(op, u))
			allTargets = append(allTargets, target)
			mf.Rules = append(mf.Rules, &makex.BasicRule{
				TargetFile:  target,
				PrereqFiles: []string{filepath.Join(filepath.Dir(buildDataDir), unitFile)},
				RecipeCmds:  []string{fmt.Sprintf("src tool %q %q < $^ 1> $@", toolRef.Toolchain, toolRef.Subcmd)},
			})
		}
	}
	mf.Rules = append(mf.Rules, &makex.BasicRule{
		TargetFile:  "all",
		PrereqFiles: allTargets,
	})

	mfData, err := makex.Marshal(&mf)
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
