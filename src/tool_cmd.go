package src

import (
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/sqs/go-flags"
	"sourcegraph.com/sourcegraph/srclib/toolchain"
)

func init() {
	c, err := CLI.AddCommand("tool",
		"run a tool",
		"Run a srclib tool with the specified arguments.",
		&toolCmd,
	)
	if err != nil {
		log.Fatal(err)
	}
	c.ArgsRequired = true
}

type ToolCmd struct {
	ToolchainExecOpt

	Args struct {
		Toolchain ToolchainPath `name:"TOOLCHAIN" description:"toolchain path of the toolchain to run"`
		Tool      ToolName      `name:"TOOL" description:"tool subcommand name to run (in TOOLCHAIN)"`
		ToolArgs  []string      `name:"ARGS" description:"args to pass to TOOL"`
	} `positional-args:"yes" required:"yes"`
}

var toolCmd ToolCmd

func (c *ToolCmd) Execute(args []string) error {
	tc, err := toolchain.Open(string(c.Args.Toolchain), c.ToolchainMode())
	if err != nil {
		log.Fatal(err)
	}

	var cmder interface {
		Command() (*exec.Cmd, error)
	}
	if c.Args.Tool != "" {
		cmder, err = toolchain.OpenTool(string(c.Args.Toolchain), string(c.Args.Tool), c.ToolchainMode())
	} else {
		cmder = tc
	}

	cmd, err := cmder.Command()
	if err != nil {
		log.Fatal(err)
	}

	cmd.Args = append(cmd.Args, c.Args.ToolArgs...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	if GlobalOpt.Verbose {
		log.Printf("Running tool: %v", cmd.Args)
	}
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}

	return nil
}

type ToolName string

func (t ToolName) Complete(match string) []flags.Completion {
	// Assume toolchain is the last arg.
	toolchainPath := os.Args[len(os.Args)-2]
	tc, err := toolchain.Lookup(toolchainPath)
	if err != nil {
		log.Println(err)
		return nil
	}
	c, err := tc.ReadConfig()
	if err != nil {
		log.Println(err)
		return nil
	}
	var comps []flags.Completion
	for _, tt := range c.Tools {
		if strings.HasPrefix(tt.Subcmd, match) {
			comps = append(comps, flags.Completion{Item: tt.Subcmd})
		}
	}
	return comps
}
