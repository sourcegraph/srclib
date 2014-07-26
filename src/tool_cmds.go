package src

import (
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/jessevdk/go-flags"
	"github.com/sourcegraph/srclib/toolchain"
)

func init() {
	c, err := parser.AddCommand("tool",
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
	ExeMethods   string `short:"m" long:"methods" default:"program,docker" description:"permitted execution methods" value-name:"METHODS"`
	ForceRebuild bool   `short:"b" long:"rebuild" description:"force rebuild of Docker image"`

	Args struct {
		Toolchain ToolchainPath `name:"TOOLCHAIN" description:"toolchain path of the toolchain to run"`
		Tool      ToolName      `name:"TOOL" description:"tool subcommand name to run (in TOOLCHAIN)"`
		ToolArgs  []string      `name:"ARGS" description:"args to pass to TOOL"`
	} `positional-args:"yes" required:"yes"`
}

var toolCmd ToolCmd

func (c *ToolCmd) Execute(args []string) error {
	mode := parseExeMethods(c.ExeMethods)

	tc, err := toolchain.Open(string(c.Args.Toolchain), mode)
	if err != nil {
		log.Fatal(err)
	}
	if c.ForceRebuild {
		if err := tc.Build(); err != nil {
			log.Fatal(err)
		}
	}

	var cmder interface {
		Command() (*exec.Cmd, error)
	}
	if c.Args.Tool != "" {
		cmder, err = toolchain.OpenTool(string(c.Args.Toolchain), string(c.Args.Tool), mode)
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
	if gopt.Verbose {
		log.Printf("Running tool: %v", cmd.Args)
	}
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}

	return nil
}

type ToolchainPath string

func (t ToolchainPath) Complete(match string) []flags.Completion {
	toolchains, err := toolchain.List()
	if err != nil {
		log.Println(err)
		return nil
	}
	var comps []flags.Completion
	for _, tc := range toolchains {
		if strings.HasPrefix(tc.Path, match) {
			comps = append(comps, flags.Completion{Item: tc.Path})
		}
	}
	return comps
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
	tools, err := tc.Tools()
	if err != nil {
		log.Println(err)
		return nil
	}
	var comps []flags.Completion
	for _, tt := range tools {
		if strings.HasPrefix(tt.Subcmd, match) {
			comps = append(comps, flags.Completion{Item: tt.Subcmd})
		}
	}
	return comps
}
