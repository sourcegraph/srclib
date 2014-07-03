package srcgraph

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"

	"sourcegraph.com/sourcegraph/conf"
	"sourcegraph.com/sourcegraph/srcgraph/client"
	"sourcegraph.com/sourcegraph/srcgraph/task2"
)

var (
	Name      = "srcgraph"
	ExtraHelp = ""
)

func init() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, Name+` builds projects for and queries Sourcegraph.
`+ExtraHelp+`
Usage:

        `+Name+` [options] command [arg...]

The commands are:
`)
		for _, c := range Subcommands {
			fmt.Fprintf(os.Stderr, "    %-24s %s\n", c.Name, c.Description)
		}
		fmt.Fprintln(os.Stderr, `
Use "`+Name+` command -h" for more information about a command.

The options are:
`)
		flag.PrintDefaults()
		os.Exit(1)
	}
}

var Verbose = flag.Bool("v", false, "show verbose output")
var Dir = flag.String("dir", ".", "directory to work in")

var apiclient = client.NewClient(nil)

func init() {
	apiclient.BaseURL = conf.BaseURL.ResolveReference(&url.URL{Path: "/api/"})
}

func Main() {
	flag.Parse()

	if flag.NArg() == 0 {
		flag.Usage()
	}
	log.SetFlags(0)
	log.SetPrefix("")
	defer task2.FlushAll()

	subcmd := flag.Arg(0)
	extraArgs := flag.Args()[1:]
	if subcmd == "help" {
		help(extraArgs)
	} else {
		for _, c := range Subcommands {
			if c.Name == subcmd {
				c.Run(extraArgs)
				return
			}
		}
	}

	fmt.Fprintf(os.Stderr, Name+": unknown subcommand %q\n", subcmd)
	fmt.Fprintln(os.Stderr, `Run "`+Name+` -h" for usage.`)
	os.Exit(1)
}

type Subcommand struct {
	Name        string
	Description string
	Run         func(args []string)
}

var Subcommands = []Subcommand{
	{"make", "make a repository", make_},
	{"makefile", "print the Makefile and exit", makefile},
	{"scan", "scan a repository for source units", scan_},
	{"config", "validate and print a repository's configuration", config_},
	{"list-deps", "list a repository's raw (unresolved) dependencies", listDeps},
	{"resolve-deps", "resolve a repository's raw dependencies", resolveDeps},
	{"graph", "analyze a repository's source code for definitions and references", graph_},
	{"blame", "blame a source unit's source files to determine commit authors", blame},
	{"authorship", "determine authorship of a source unit's symbols and refs", authorship_},
	{"person-refresh-profile", "refresh a person's profile", personRefreshProfile},
	{"person-compute-stats", "update a person's stats", personComputeStats},
	{"repo-create", "create a repository (API)", repoCreate},
	{"repo-refresh-profile", "sync repository data", repoRefreshProfile},
	{"repo-refresh-vcs-data", "fetch repository VCS data", repoRefreshVCSData},
	{"repo-compute-stats", "update and print repository stats", repoComputeStats},
	{"build", "create a new build for a repository (API)", build_},
	{"build-queue", "display the build queue (API)", buildQueue},
	{"info", "show info about enabled capabilities", info},
	{"help", "show help about a command", nil},
}

func help(args []string) {
	fs := flag.NewFlagSet("help", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: `+Name+` help command

Shows information about a `+Name+` command.

The options are:
`)
		fs.PrintDefaults()
		os.Exit(1)
	}
	fs.Parse(args)

	if fs.NArg() != 1 {
		fs.Usage()
	}

	subcmd := fs.Arg(0)
	for _, c := range Subcommands {
		if c.Name == subcmd {
			c.Run([]string{"-h"})
			return
		}
	}
}
