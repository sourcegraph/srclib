package src

import (
	"fmt"
	"log"

	"strings"

	"github.com/sourcegraph/srclib/toolchain"
	"github.com/sqs/go-flags"
)

func init() {
	c, err := CLI.AddCommand("toolchain",
		"manage toolchains",
		"Manage srclib toolchains.",
		&toolchainCmd,
	)
	if err != nil {
		log.Fatal(err)
	}

	_, err = c.AddCommand("list",
		"list available toolchains",
		"List available toolchains.",
		&toolchainListCmd,
	)
	if err != nil {
		log.Fatal(err)
	}

	_, err = c.AddCommand("list-tools",
		"list tools in toolchains",
		"List available tools in all toolchains.",
		&toolchainListToolsCmd,
	)
	if err != nil {
		log.Fatal(err)
	}

	_, err = c.AddCommand("build",
		"build a toolchain",
		"Build a toolchain's Docker image.",
		&toolchainBuildCmd,
	)
	if err != nil {
		log.Fatal(err)
	}

	_, err = c.AddCommand("get",
		"download a toolchain",
		"Download a toolchain's repository to the SRCLIBPATH.",
		&toolchainGetCmd,
	)
	if err != nil {
		log.Fatal(err)
	}

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

type ToolchainExecOpt struct {
	ExeMethods string `short:"m" long:"methods" default:"program,docker" description:"permitted execution methods" value-name:"METHODS"`
}

func (o *ToolchainExecOpt) ToolchainMode() toolchain.Mode {
	// TODO(sqs): make this a go-flags type
	methods := strings.Split(o.ExeMethods, ",")
	var mode toolchain.Mode
	for _, method := range methods {
		if method == "program" {
			mode |= toolchain.AsProgram
		}
		if method == "docker" {
			mode |= toolchain.AsDockerContainer
		}
	}
	return mode
}

type ToolchainCmd struct{}

var toolchainCmd ToolchainCmd

func (c *ToolchainCmd) Execute(args []string) error { return nil }

type ToolchainListCmd struct {
}

var toolchainListCmd ToolchainListCmd

func (c *ToolchainListCmd) Execute(args []string) error {
	toolchains, err := toolchain.List()
	if err != nil {
		return err
	}

	fmtStr := "%-40s  %s\n"
	fmt.Printf(fmtStr, "PATH", "TYPE")
	for _, t := range toolchains {
		var exes []string
		if t.Program != "" {
			exes = append(exes, "program")
		}
		if t.Dockerfile != "" {
			exes = append(exes, "docker")
		}
		fmt.Printf(fmtStr, t.Path, strings.Join(exes, ", "))
	}
	return nil
}

type ToolchainListToolsCmd struct {
	Args struct {
		Toolchains []ToolchainPath `name:"TOOLCHAINS" description:"list tools in these toolchains only"`
	} `positional-args:"yes" required:"yes"`
}

var toolchainListToolsCmd ToolchainListToolsCmd

func (c *ToolchainListToolsCmd) Execute(args []string) error {
	tcs, err := toolchain.List()
	if err != nil {
		log.Fatal(err)
	}

	fmtStr := "%-18s  %-15s  %-40s\n"
	fmt.Printf(fmtStr, "TOOL", "OP", "TOOLCHAIN")
	for _, tc := range tcs {
		if len(c.Args.Toolchains) > 0 {
			found := false
			for _, tc2 := range c.Args.Toolchains {
				if string(tc2) == tc.Path {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		tools, err := tc.Tools()
		if err != nil {
			log.Fatal(err)
		}
		for _, t := range tools {
			fmt.Printf(fmtStr, t.Subcmd, t.Op, t.Toolchain.Path)
		}
	}
	return nil
}

type ToolchainBuildCmd struct {
	Args struct {
		Toolchains []ToolchainPath `name:"TOOLCHAINS" description:"toolchain paths of toolchains to build"`
	} `positional-args:"yes" required:"yes"`
}

var toolchainBuildCmd ToolchainBuildCmd

func (c *ToolchainBuildCmd) Execute(args []string) error {
	for _, tc := range c.Args.Toolchains {
		tc, err := toolchain.Open(string(tc), toolchain.AsDockerContainer)
		if err != nil {
			log.Fatal(err)
		}
		if err := tc.Build(); err != nil {
			log.Fatal(err)
		}
	}
	return nil
}

type ToolchainGetCmd struct {
	Args struct {
		Toolchains []ToolchainPath `name:"TOOLCHAINS" description:"toolchain paths of toolchains to get"`
	} `positional-args:"yes" required:"yes"`
}

var toolchainGetCmd ToolchainGetCmd

func (c *ToolchainGetCmd) Execute(args []string) error {
	for _, tc := range c.Args.Toolchains {
		if gopt.Verbose {
			fmt.Println(tc)
		}
		_, err := toolchain.Get(string(tc))
		if err != nil {
			return err
		}
	}
	return nil
}
