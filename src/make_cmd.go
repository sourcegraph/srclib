package src

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/sourcegraph/srclib/buildstore"
	"github.com/sourcegraph/srclib/config"
)

func init() {
	c, err := CLI.AddCommand("make",
		"build a tree (config, plan, and execute)",
		`Configures a tree, plans the execution, and executes all analysis steps.`,
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

	// config
	configCmd := &ConfigCmd{
		Options:          c.Options,
		ToolchainExecOpt: c.ToolchainExecOpt,
		BuildCacheOpt:    c.BuildCacheOpt,
	}
	if err := configCmd.Execute(nil); err != nil {
		return err
	}

	// plan
	planCmd := &PlanCmd{}
	if err := planCmd.Execute(nil); err != nil {
		return err
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
	buildRoot, err := buildstore.RootDir(buildStore)
	if err != nil {
		return err
	}
	mfFile := filepath.Join(buildRoot, buildStore.FilePath(currentRepo.CommitID, "Makefile"))
	makeCmd := exec.Command("make", "-f", mfFile)
	makeCmd.Args = append(makeCmd.Args, c.Args.Targets...)
	makeCmd.Stdout = os.Stderr
	makeCmd.Stderr = os.Stderr
	if err := makeCmd.Run(); err != nil {
		return err
	}

	return nil
}
