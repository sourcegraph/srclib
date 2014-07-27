package src

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/kr/fs"
	"github.com/sourcegraph/makex"
	"github.com/sourcegraph/srclib/buildstore"
	"github.com/sourcegraph/srclib/dep2"
	"github.com/sourcegraph/srclib/plan"
	"github.com/sourcegraph/srclib/unit"
	"github.com/sqs/go-flags"
)

func init() {
	c, err := CLI.AddCommand("plan",
		"generate a Makefile to process a project",
		`Generate a Makefile to process a repository or directory tree.

If CONFIG-FILE is "-", it is read from stdin. If no CONFIG-FILE is given, "src config --output json" is executed in the current directory and its output is used as the configuration.
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
		ConfigFile flags.Filename `name:"CONFIG-FILE" description:"project config JSON file produced by 'src config' (unset means stdin)"`
	} `positional-args:"yes"`
}

var planCmd PlanCmd

func (c *PlanCmd) Execute(args []string) error {
	// Get all .srclib-cache/**/*.unit.v0.json files.
	currentRepo, err := OpenRepo(Dir)
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

		target := filepath.Join(buildDataDir, plan.SourceUnitDataFilename([]*dep2.ResolvedDep{}, u))
		allTargets = append(allTargets, target)
		mf.Rules = append(mf.Rules, &makex.BasicRule{
			TargetFile:  target,
			PrereqFiles: []string{filepath.Join(filepath.Dir(buildDataDir), unitFile)},
			RecipeCmds:  []string{"src tool github.com/sourcegraph/srclib-go depresolve < $^ 1> $@"},
		})
	}
	mf.Rules = append(mf.Rules, &makex.BasicRule{
		TargetFile:  "all",
		PrereqFiles: allTargets,
	})

	mfData, err := makex.Marshal(&mf)
	if err != nil {
		log.Fatal(err)
	}
	os.Stdout.Write(mfData)

	return nil
}
