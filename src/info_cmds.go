package src

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/sourcegraph/srclib/build"
	"github.com/sourcegraph/srclib/buildstore"
	"github.com/sourcegraph/srclib/toolchain"
)

func infoCmd(args []string) {
	fs := flag.NewFlagSet("info", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: `+Name+` info [toolchains|tools]

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
		case "toolchains":
			toolchainsCmd(extraArgs)
		case "tools":
			toolsCmd(extraArgs)
		default:
			log.Fatalf("No info on %q.", what)
		}
		return
	}

	log.Printf("srclib version %s\n", Version)
	log.Println("https://sourcegraph.com/sourcegraph/srclib")
	log.Println()

	log.Printf("SRCLIBPATH=%q", toolchain.SrclibPath)

	log.Println()
	log.Println("TOOLCHAINS =======================================================================")
	toolchainsCmd(nil)
	log.Println()
	log.Println()

	log.Println("TOOLS ============================================================================")
	toolsCmd(nil)

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

func toolchainsCmd(args []string) {
	fs := flag.NewFlagSet("toolchains", flag.ExitOnError)
	quiet := fs.Bool("q", false, "quiet (only show names/URIs)")
	json := fs.Bool("json", false, "print output as JSON")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: `+Name+` toolchains

Prints all available toolchains that contain a Srclibtoolchain file.

Toolchains without a Srclibtoolchain file can still be run, but they won't be appear in
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

	toolchains, err := toolchain.List()
	if err != nil {
		log.Fatal(err)
	}

	fmtStr := "%-60s  %s\n"
	if !*quiet && !*json {
		fmt.Printf(fmtStr, "PATH", "TYPE")
	}
	for _, t := range toolchains {
		if *quiet {
			fmt.Println(t.Path)
		} else if *json {
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
}

func toolsCmd(args []string) {
	fs := flag.NewFlagSet("tools", flag.ExitOnError)
	quiet := fs.Bool("q", false, "quiet (only show tool subcommands, no toolchain names; use with -toolchain)")
	common := fs.Bool("common", false, "show all subcommands (even non-tool subcommands like 'version' and 'help')")
	json := fs.Bool("json", false, "print output as JSON")
	toolchainPath := fs.String("toolchain", "", "only show this toolchain's tools")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: `+Name+` tools [opts]

Prints all tools implemented by the available toolchains.

The options are:
`)
		fs.PrintDefaults()
		os.Exit(1)
	}
	fs.Parse(args)

	if fs.NArg() != 0 {
		fs.Usage()
	}

	toolchains, err := toolchain.List()
	if err != nil {
		log.Fatal(err)
	}

	fmtStr := "%-12s  %-60s\n"
	if !*quiet && !*json {
		fmt.Printf(fmtStr, "TOOL", "TOOLCHAIN")
	}
	for _, t := range toolchains {
		if *toolchainPath != "" && t.Path != *toolchainPath {
			continue
		}
		tools, err := t.Tools()
		if err != nil {
			log.Fatal(err)
		}
		for _, t := range tools {
			if _, isCommon := toolchain.CommonSubcommands[t.Subcmd]; isCommon && !*common {
				continue
			}
			if *quiet {
				fmt.Println(t.Subcmd)
			} else if *json {
				PrintJSON(t, "")
			} else {
				fmt.Printf(fmtStr, t.Subcmd, t.Toolchain.Path)
			}
		}
	}
}
