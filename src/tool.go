package src

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/sourcegraph/srclib/toolchain"
)

func toolsCmd(args []string) {
	fs := flag.NewFlagSet("tools", flag.ExitOnError)
	quiet := fs.Bool("q", false, "quiet (only show names/URIs)")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: `+Name+` tools

Prints all available tools that contain a Srclibtool file.

Tools without a Srclibtool file can still be run, but they won't be appear in
this list.

The options are:
`)
		fs.PrintDefaults()
		os.Exit(1)
	}
	fs.Parse(args)

	if fs.NArg() != 0 {
		fs.Usage()
	}

	tools, err := toolchain.FindAll()
	if err != nil {
		log.Fatal(err)
	}

	for _, t := range tools {
		if *quiet {
			fmt.Println(t.Name())
		} else {
			fmt.Printf("%-60s  %s\n", t.Name(), t.Type())
		}
	}
}

func toolCmd(args []string) {
	fs := flag.NewFlagSet("tool", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: `+Name+` tool TOOL [ARG...]

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

	tool, err := toolchain.Lookup(toolName)
	if err != nil {
		log.Fatal(err)
	}

	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
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
