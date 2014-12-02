package src

import (
	"log"
	"os"
	"path/filepath"

	"sourcegraph.com/sourcegraph/makex"

	"strings"

	"sourcegraph.com/sourcegraph/srclib/buildstore"
	"sourcegraph.com/sourcegraph/srclib/config"
	"sourcegraph.com/sourcegraph/srclib/plan"
	"sourcegraph.com/sourcegraph/srclib/toolchain"
)

func init() {
	c, err := CLI.AddCommand("make",
		"plans and executes plan",
		`Generates a plan (in Makefile form, in memory) for analyzing the tree and executes the plan. `,
		&makeCmd,
	)
	if err != nil {
		log.Fatal(err)
	}

	SetRepoOptDefaults(c)
}

type MakeCmd struct {
	config.Options

	ToolchainExecOpt `group:"execution"`
	BuildCacheOpt    `group:"build cache"`

	PrintMakefile bool `short:"p" long:"print" description:"print planned Makefile and exit"`
	DryRun        bool `short:"n" long:"dry-run" description:"print what would be done and exit"`

	Dir Directory `short:"C" long:"directory" description:"change to DIR before doing anything" value-name:"DIR"`

	Args struct {
		Goals []string `name:"GOALS..." description:"Makefile targets to build (default: all)"`
	} `positional-args:"yes"`
}

var makeCmd MakeCmd

func (c *MakeCmd) Execute(args []string) error {
	if c.Dir != "" {
		if err := os.Chdir(string(c.Dir)); err != nil {
			return err
		}
	}

	mk, mf, err := CreateMaker(c.ToolchainExecOpt, c.Args.Goals)
	if err != nil {
		return err
	}

	if c.PrintMakefile {
		mfData, err := makex.Marshal(mf)
		if err != nil {
			return err
		}
		_, err = os.Stdout.Write(mfData)
		return err
	}

	if c.DryRun {
		return mk.DryRun(os.Stdout)
	}

	return mk.Run()
}

// CreateMaker creates a Makefile and a Maker. The cwd should be the root of the
// tree you want to make (due to some probably unnecessary assumptions that
// CreateMaker makes).
func CreateMaker(execOpt ToolchainExecOpt, goals []string) (*makex.Maker, *makex.Makefile, error) {
	currentRepo, err := OpenRepo(".")
	if err != nil {
		return nil, nil, err
	}
	buildStore, err := buildstore.NewRepositoryStore(currentRepo.RootDir)
	if err != nil {
		return nil, nil, err
	}

	treeConfig, err := config.ReadCached(buildStore, currentRepo.CommitID)
	if err != nil {
		return nil, nil, err
	}
	if len(treeConfig.SourceUnits) == 0 {
		log.Println("No source unit files found. Did you mean to run `src config`? (This is not an error; it just means that src didn't find anything to build or analyze here.)")
	}

	toolchainExecOptArgs, err := toolchain.MarshalArgs(&execOpt)
	if err != nil {
		return nil, nil, err
	}

	buildDataDir, err := buildstore.BuildDir(buildStore, currentRepo.CommitID)
	if err != nil {
		return nil, nil, err
	}
	absDir, err := os.Getwd()
	if err != nil {
		return nil, nil, err
	}
	buildDataDir, _ = filepath.Rel(absDir, buildDataDir)

	mf, err := plan.CreateMakefile(buildDataDir, treeConfig, plan.Options{ToolchainExecOpt: strings.Join(toolchainExecOptArgs, " ")})
	if err != nil {
		return nil, nil, err
	}

	if len(goals) == 0 {
		if defaultRule := mf.DefaultRule(); defaultRule != nil {
			goals = []string{defaultRule.Target()}
		}
	}

	mkConf := &makex.Default
	mk := mkConf.NewMaker(mf, goals...)

	return mk, mf, nil
}
