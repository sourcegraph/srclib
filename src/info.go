package src

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/sourcegraph/srclib/build"
	"github.com/sourcegraph/srclib/buildstore"
	"github.com/sourcegraph/srclib/tool"
)

func infoCmd(args []string) {
	fs := flag.NewFlagSet("info", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: `+Name+` info [tools|handlers]

Shows information about enabled capabilities in this tool as well as system
information.

The options are:
`)
		fs.PrintDefaults()
		os.Exit(1)
	}
	fs.Parse(args)

	if fs.NArg() > 0 {
		extraArgs := fs.Args()[1:]
		what := fs.Arg(0)
		switch what {
		case "tools":
			toolsCmd(extraArgs)
		case "handlers":
			handlersCmd(extraArgs)
		default:
			log.Fatalf("No info on %q.", what)
		}
		return
	}

	log.Printf("srclib version %s\n", Version)
	log.Println("https://sourcegraph.com/sourcegraph/srclib")
	log.Println()

	log.Printf("SRCLIBPATH=%q", tool.SrclibPath)

	log.Println()
	log.Println("TOOLS ==========================================================================")
	toolsCmd(nil)
	log.Println()
	log.Println()

	log.Println("HANDLERS ============================================================================")
	handlersCmd(nil)

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
}

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

	tools, err := tool.List()
	if err != nil {
		log.Fatal(err)
	}

	fmtStr := "%-60s  %s\n"
	if !*quiet {
		fmt.Printf(fmtStr, "NAME", "TYPE")
	}
	for _, t := range tools {
		if *quiet {
			fmt.Println(t.Name())
		} else {
			fmt.Printf(fmtStr, t.Name(), t.Type())
		}
	}
}

func handlersCmd(args []string) {
	fs := flag.NewFlagSet("handlers", flag.ExitOnError)
	quiet := fs.Bool("q", false, "quiet (only show handler names, no tool names)")
	common := fs.Bool("common", false, "show all handlers (even common subcommands like 'version' and 'help')")
	toolName := fs.String("tool", "", "only show this tool's handlers")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: `+Name+` handlers [opts]

Prints all handlers implemented by the available tools.

Handlers provided by tools without a Srclibtool file can still be run, but
they won't be appear in this list.

The options are:
`)
		fs.PrintDefaults()
		os.Exit(1)
	}
	fs.Parse(args)

	if fs.NArg() != 0 {
		fs.Usage()
	}

	tools, err := tool.List()
	if err != nil {
		log.Fatal(err)
	}

	fmtStr := "%-12s  %-60s\n"
	if !*quiet {
		fmt.Printf(fmtStr, "HANDLER", "TOOL")
	}
	for _, t := range tools {
		if *toolName != "" && t.Name() != *toolName {
			continue
		}
		handlers, err := t.Handlers()
		if err != nil {
			log.Fatal(err)
		}
		for _, h := range handlers {
			if _, isCommon := tool.CommonSubcommands[h]; isCommon && !*common {
				continue
			}
			if *quiet {
				fmt.Println(h)
			} else {
				fmt.Printf(fmtStr, h, t.Name())
			}
		}
	}
}
