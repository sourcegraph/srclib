package src

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/sourcegraph/srclib/toolchain"
)

func toolCmd(args []string) {
	fs := flag.NewFlagSet("tool", flag.ExitOnError)
	forceRebuild := fs.Bool("b", false, "force rebuild of Docker image")
	exeMethods := fs.String("m", defaultExeMethods, "permitted execution methods")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: `+Name+` tool [OPT] TOOLCHAIN TOOL [ARG...]

Run a srclib tool with the specified arguments.

The options are:
`)
		fs.PrintDefaults()
		os.Exit(1)
	}
	fs.Parse(args)

	if fs.NArg() == 0 {
		fs.Usage()
	}

	var toolSubcmd string
	var toolArgs []string

	toolchainPath := fs.Arg(0)
	if fs.NArg() > 1 {
		a := fs.Arg(1)
		if len(a) > 0 && a[0] != '-' {
			toolSubcmd = a
			toolArgs = fs.Args()[2:]
		} else {
			toolArgs = fs.Args()[1:]
		}
	}
	mode := parseExeMethods(*exeMethods)

	tc, err := toolchain.Open(toolchainPath, mode)
	if err != nil {
		log.Fatal(err)
	}
	if *forceRebuild {
		if err := tc.Build(); err != nil {
			log.Fatal(err)
		}
	}

	var cmder interface {
		Command() (*exec.Cmd, error)
	}
	if toolSubcmd != "" {
		cmder, err = toolchain.OpenTool(toolchainPath, toolSubcmd, mode)
	} else {
		cmder = tc
	}

	cmd, err := cmder.Command()
	if err != nil {
		log.Fatal(err)
	}

	cmd.Args = append(cmd.Args, toolArgs...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if *Verbose {
		log.Printf("Running tool: %v", cmd.Args)
	}
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
}
