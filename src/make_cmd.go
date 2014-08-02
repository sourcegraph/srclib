package src

import (
	"log"
	"os"
	"path/filepath"

	"github.com/sourcegraph/makex"

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
		Targets []string `name:"TARGETS..." description:"Makefile targets to build (default: all)"`
	} `positional-args:"yes"`
}

var makeCmd MakeCmd

func (c *MakeCmd) Execute(args []string) error {
	if c.Dir != "" {
		if err := os.Chdir(string(c.Dir)); err != nil {
			return err
		}
	}

	if len(c.Args.Targets) == 0 {
		c.Args.Targets = []string{"all"}
	}

	// execute
	// TODO(sqs): use makex and makefile returned by planCmd
	currentRepo, err := OpenRepo(".")
	if err != nil {
		return err
	}
	buildStore, err := buildstore.NewRepositoryStore(currentRepo.RootDir)
	if err != nil {
		return err
	}

	treeConfig, err := config.ReadCached(buildStore, currentRepo.CommitID)
	if err != nil {
		return err
	}

	toolchainExecOptArgs, err := toolchain.MarshalArgs(&c.ToolchainExecOpt)
	if err != nil {
		return err
	}

	buildDataDir, err := buildstore.BuildDir(buildStore, currentRepo.CommitID)
	if err != nil {
		return err
	}
	absDir, err := os.Getwd()
	if err != nil {
		return err
	}
	buildDataDir, _ = filepath.Rel(absDir, buildDataDir)

	if c.PrintMakefile {
		mf, err := plan.CreateMakefile(buildDataDir, treeConfig, plan.Options{strings.Join(toolchainExecOptArgs, " ")})
		if err != nil {
			return err
		}
		mfData, err := makex.Marshal(mf)
		if err != nil {
			return err
		}
		_, err = os.Stdout.Write(mfData)
		return err
	}

	mf, err := plan.CreateMakefile(buildDataDir, treeConfig, plan.Options{strings.Join(toolchainExecOptArgs, " ")})
	if err != nil {
		return err
	}

	mkConf := &makex.Default
	mk := mkConf.NewMaker(mf, c.Args.Targets...)

	if c.DryRun {
		return mk.DryRun(os.Stdout)
	}

	return mk.Run()
}
