package src

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/sourcegraph/srclib/tool"
)

func toolCmd(args []string) {
	fs := flag.NewFlagSet("tool", flag.ExitOnError)
	forceRebuild := fs.Bool("b", false, "force rebuild of Docker image")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: `+Name+` tool [OPT] TOOL [ARG...]

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

	toolName, toolArgs := fs.Arg(0), fs.Args()[1:]

	tool, err := tool.Lookup(toolName)
	if err != nil {
		log.Fatal(err)
	}

	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	if *forceRebuild {
		if err := tool.Build(); err != nil {
			log.Fatal(err)
		}
	}

	cmd, err := tool.Command(dir)
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
