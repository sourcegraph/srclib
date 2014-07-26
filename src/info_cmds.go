package src

import (
	"fmt"
	"log"
	"strings"

	"github.com/sourcegraph/srclib/build"
	"github.com/sourcegraph/srclib/buildstore"
	"github.com/sourcegraph/srclib/toolchain"
)

func init() {
	c, err := parser.AddCommand("info",
		"show info about enabled capabilities",
		"Shows information about enabled capabilities in this tool as well as system information.",
		&infoCmd,
	)
	if err != nil {
		log.Fatal(err)
	}

	c.SubcommandsOptional = true

	_, err = c.AddCommand("toolchains",
		"list available toolchains",
		"Prints all available toolchains that contain a Srclibtoolchain file.",
		&infoToolchainsCmd,
	)
	if err != nil {
		log.Fatal(err)
	}

	_, err = c.AddCommand("tools",
		"list available tools",
		"Prints all available tools in toolchains.",
		&infoToolsCmd,
	)
	if err != nil {
		log.Fatal(err)
	}
}

type InfoCmd struct{}

var infoCmd InfoCmd

func (c *InfoCmd) Execute(args []string) error {
	log.Printf("srclib version %s\n", Version)
	log.Println("https://sourcegraph.com/sourcegraph/srclib")
	log.Println()

	log.Printf("SRCLIBPATH=%q", toolchain.SrclibPath)

	log.Println()
	log.Printf("Build data types (%d)", len(buildstore.DataTypes))
	for name, _ := range buildstore.DataTypes {
		log.Printf(" - %s", name)
	}
	log.Println()

	log.Printf("Build rule makers (%d)", len(build.RuleMakers))
	for name, _ := range build.RuleMakers {
		log.Printf(" - %s", name)
	}

	return nil
}

type InfoToolchainsCmd struct {
	Quiet bool `short:"q"`
	JSON  bool `long:"json" description:"print output as JSON"`
}

var infoToolchainsCmd InfoToolchainsCmd

func (c *InfoToolchainsCmd) Execute(args []string) error {
	toolchains, err := toolchain.List()
	if err != nil {
		log.Fatal(err)
	}

	fmtStr := "%-60s  %s\n"
	if !c.Quiet && !c.JSON {
		fmt.Printf(fmtStr, "PATH", "TYPE")
	}
	for _, t := range toolchains {
		if c.Quiet {
			fmt.Println(t.Path)
		} else if c.JSON {
			PrintJSON(t, "")
		} else {
			var exes []string
			if t.Program != "" {
				exes = append(exes, "program")
			}
			if t.Dockerfile != "" {
				exes = append(exes, "docker")
			}
			fmt.Printf(fmtStr, t.Path, strings.Join(exes, ", "))
		}
	}

	return nil
}

type InfoToolsCmd struct {
	Quiet         bool   `short:"q" description:"quiet (only show tool subcommands, no toolchain names; use with -toolchain)"`
	JSON          bool   `long:"json" description:"print output as JSON"`
	ToolchainPath string `long:"toolchain" description:"show only this toolchain's tools" value-name:"PATH"`
	Common        bool   `long:"common" description:"show all subcommands (even non-tool subcommands like 'version' and 'help')"`
}

var infoToolsCmd InfoToolsCmd

func (c *InfoToolsCmd) Execute(args []string) error {
	tools, err := toolchain.List()
	if err != nil {
		log.Fatal(err)
	}

	fmtStr := "%-12s  %-60s\n"
	if !c.Quiet && !c.JSON {
		fmt.Printf(fmtStr, "TOOL", "TOOLCHAIN")
	}
	for _, t := range tools {
		if c.ToolchainPath != "" && t.Path != c.ToolchainPath {
			continue
		}
		tools, err := t.Tools()
		if err != nil {
			log.Fatal(err)
		}
		for _, t := range tools {
			if _, isCommon := toolchain.CommonSubcommands[t.Subcmd]; isCommon && c.Common {
				continue
			}
			if c.Quiet {
				fmt.Println(t.Subcmd)
			} else if c.JSON {
				PrintJSON(t, "")
			} else {
				fmt.Printf(fmtStr, t.Subcmd, t.Toolchain.Path)
			}
		}
	}

	return nil
}
