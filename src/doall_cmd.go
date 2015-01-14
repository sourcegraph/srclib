package src

import (
	"log"
	"os"

	"sourcegraph.com/sourcegraph/srclib/config"
)

func init() {
	// TODO(sqs): "do-all" is a stupid name
	c, err := CLI.AddCommand("do-all",
		"fully process (config, plan, execute, and import)",
		`Fully processes a tree: configures it, plans the execution, executes all analysis steps, and imports the data.`,
		&doAllCmd,
	)
	if err != nil {
		log.Fatal(err)
	}

	setDefaultRepoURIOpt(c)
	setDefaultRepoSubdirOpt(c)
}

type DoAllCmd struct {
	config.Options

	ToolchainExecOpt `group:"execution"`
	BuildCacheOpt    `group:"build cache"`

	Dir Directory `short:"C" long:"directory" description:"change to DIR before doing anything" value-name:"DIR"`
}

var doAllCmd DoAllCmd

func (c *DoAllCmd) Execute(args []string) error {
	if c.Dir != "" {
		if err := os.Chdir(string(c.Dir)); err != nil {
			return err
		}
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

	// make
	makeCmd := &MakeCmd{
		Options:          c.Options,
		ToolchainExecOpt: c.ToolchainExecOpt,
		BuildCacheOpt:    c.BuildCacheOpt,
	}
	if err := makeCmd.Execute(nil); err != nil {
		return err
	}

	// store
	storeCmd := &StoreCmd{
		Options: c.Options,
	}
	if err := storeCmd.Execute(nil); err != nil {
		return err
	}

	return nil
}
