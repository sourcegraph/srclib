package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sourcegraph/go-vcs"
	"sourcegraph.com/sourcegraph/config2"
	"sourcegraph.com/sourcegraph/repo"
	"sourcegraph.com/sourcegraph/srcgraph/build"
	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/dep2"
	"sourcegraph.com/sourcegraph/srcgraph/grapher2"
	"sourcegraph.com/sourcegraph/srcgraph/scan"
	"sourcegraph.com/sourcegraph/srcgraph/task2"
	task_ui "sourcegraph.com/sourcegraph/srcgraph/task2/ui"
)

var verbose = flag.Bool("v", false, "show verbose output")
var dir = flag.String("dir", ".", "directory to work in")
var tmpDir = flag.String("tmpdir", filepath.Join(os.TempDir(), "sg"), "temporary directory to use")

func main() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, `srcgraph builds projects for and queries Sourcegraph.

Usage:

        srcgraph [options] command [arg...]

The commands are:
`)
		for _, c := range subcommands {
			fmt.Fprintf(os.Stderr, "    %-10s %s\n", c.name, c.description)
		}
		fmt.Fprintln(os.Stderr, `
Use "srcgraph command -h" for more information about a command.

The options are:
`)
		flag.PrintDefaults()
		os.Exit(1)
	}

	flag.Parse()

	if flag.NArg() == 0 {
		flag.Usage()
	}
	log.SetFlags(0)
	log.SetPrefix("")

	subcmd := flag.Arg(0)
	for _, c := range subcommands {
		if c.name == subcmd {
			c.run(flag.Args()[1:])
			return
		}
	}

	fmt.Fprintf(os.Stderr, "srcgraph: unknown subcommand %q\n", subcmd)
	fmt.Fprintln(os.Stderr, `Run "srcgraph -h" for usage.`)
	os.Exit(1)
}

type subcommand struct {
	name        string
	description string
	run         func(args []string)
}

var subcommands = []subcommand{
	{"build", "build a repository", build_},
	{"upload", "upload a previously generated build", upload},
	{"push", "update a repository and related information on Sourcegraph", push},
	{"scan", "scan a repository for source units", scan_},
	{"config", "validate and print a repository's configuration", config_},
	{"dep", "list and resolve a repository's dependencies", dep_},
	{"graph", "analyze a repository's source code for definitions and references", graph_},
}

func build_(args []string) {
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	repo := addRepositoryFlags(fs)
	dryRun := fs.Bool("n", false, "dry run (scans the repository and just prints out what analysis tasks would be performed)")
	ui := fs.Bool("ui", true, "use terminal UI to display job statuses")
	outputFile := fs.String("o", "", "write output to file")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: srcgraph build [options]

Builds a graph of definitions, references, and dependencies in a repository's
code at a specific revision.

The options are:
`)
		fs.PrintDefaults()
		os.Exit(1)
	}
	fs.Parse(args)

	if *outputFile == "" {
		*outputFile = repo.outputFile()
	}

	build.WorkDir = *tmpDir
	mkTmpDir()

	if fs.NArg() != 0 {
		fs.Usage()
	}

	vcsType := vcs.VCSByName[repo.vcsTypeName]
	if vcsType == nil {
		log.Fatalf("srcgraph: unknown VCS type %q", repo.vcsTypeName)
	}

	x := task2.NewRecordedContext()

	tasks, rd, err := build.Repository(repo.rootDir, repo.commitID, repo.cloneURL, vcsType, x)
	if err != nil {
		log.Fatalf("build failed: %s", err)
	}

	if *dryRun {
		task_ui.List(tasks)
		return
	}

	w := task2.Run(tasks)
	task_ui.Start(*ui, tasks, x)
	w.Wait()
	task2.FlushAll()

	f, err := os.Create(*outputFile)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	err = json.NewEncoder(f).Encode(rd)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Wrote graph output to %s.", *outputFile)
}

func upload(args []string) {
	fs := flag.NewFlagSet("upload", flag.ExitOnError)
	repo := detectRepository(*dir)
	outputFile := fs.String("f", repo.outputFile(), "graph output file to upload")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: srcgraph upload [options]

Uploads a previously generated graph to Sourcegraph.

The options are:
`)
		fs.PrintDefaults()
		os.Exit(1)
	}
	fs.Parse(args)

	if fs.NArg() != 0 {
		fs.Usage()
	}

	f, err := os.Open(*outputFile)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	var rd build.RepositoryData
	err = json.NewDecoder(f).Decode(&rd)
	if err != nil {
		log.Fatal(err)
	}

	graphData, err := json.Marshal(rd.Graph)
	if err != nil {
		log.Fatal(err)
	}

	url := config2.BaseAPIURL.ResolveReference(&url.URL{
		Path: fmt.Sprintf("repositories/%s/commits/%s/data/graph", rd.Config.URI, rd.CommitID),
	})
	req, err := http.NewRequest("PUT", url.String(), bytes.NewReader(graphData))
	if err != nil {
		log.Fatal(err)
	}
	kb := float64(len(graphData)) / 1024
	if *verbose {
		log.Printf("Uploading graph data (%.1fkb) to %s", kb, url)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		log.Fatalf("Upload failed: HTTP %s (%s).", resp.Status, string(body))
	}

	log.Printf("Uploaded graph data (%.1fkb) to %s", kb, url)
}

func push(args []string) {
	fs := flag.NewFlagSet("push", flag.ExitOnError)
	r := addRepositoryFlags(fs)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: srcgraph push [options]

Updates a repository and related information on Sourcegraph. Graph data for this
repository and commit that was previously uploaded using the "srcgraph" tool
will be used; if none exists, Sourcegraph will build the repository remotely.

The options are:
`)
		fs.PrintDefaults()
		os.Exit(1)
	}
	fs.Parse(args)

	url := config2.BaseAPIURL.ResolveReference(&url.URL{
		Path: fmt.Sprintf("repositories/%s/commits/%s/build", repo.MakeURI(r.cloneURL), r.commitID),
	})
	req, err := http.NewRequest("PUT", url.String(), nil)
	if err != nil {
		log.Fatal(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		log.Fatalf("Push failed: HTTP %s (%s).", resp.Status, string(body))
	}

	log.Printf("Push succeeded. The repository will be updated on Sourcegraph soon.")
}

func scan_(args []string) {
	fs := flag.NewFlagSet("scan", flag.ExitOnError)
	r := addRepositoryFlags(fs)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: srcgraph scan [options]

Scans a repository for source units.

The options are:
`)
		fs.PrintDefaults()
		os.Exit(1)
	}
	fs.Parse(args)

	x := task2.DefaultContext

	c, err := scan.ReadDirConfigAndScan(r.rootDir, repo.MakeURI(r.cloneURL), x)
	if err != nil {
		log.Fatal(err)
	}

	for _, u := range c.SourceUnits {
		fmt.Printf("## %s\n", u.ID())
		for _, p := range u.Paths() {
			fmt.Printf("  %s\n", p)
		}
		if *verbose {
			jsonStr, err := json.MarshalIndent(u, "\t", "  ")
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(string(jsonStr))
		}
	}
}

func config_(args []string) {
	fs := flag.NewFlagSet("config", flag.ExitOnError)
	r := addRepositoryFlags(fs)
	final := fs.Bool("final", true, "add scanned source units and finalize config before printing")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: srcgraph config [options]

Validates and prints a repository's configuration.

The options are:
`)
		fs.PrintDefaults()
		os.Exit(1)
	}
	fs.Parse(args)

	repoURI := repo.MakeURI(r.cloneURL)

	x := task2.DefaultContext

	var c *config.Repository
	var err error
	if *final {
		c, err = scan.ReadDirConfigAndScan(r.rootDir, repoURI, x)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		c, err = config.ReadDir(r.rootDir, repoURI)
		if err != nil {
			log.Fatal(err)
		}
	}

	printJSON(c, "")
}

func dep_(args []string) {
	fs := flag.NewFlagSet("dep", flag.ExitOnError)
	r := addRepositoryFlags(fs)
	resolve := fs.Bool("resolve", true, "resolve dependencies")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: srcgraph dep [options]

Lists and resolves a repository's dependencies.

The options are:
`)
		fs.PrintDefaults()
		os.Exit(1)
	}
	fs.Parse(args)

	repoURI := repo.MakeURI(r.cloneURL)

	x := task2.DefaultContext

	c, err := scan.ReadDirConfigAndScan(r.rootDir, repoURI, x)
	if err != nil {
		log.Fatal(err)
	}

	for _, u := range c.SourceUnits {
		rawDeps, err := dep2.List(r.rootDir, u, c, x)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("## ", u.ID())

		for _, rawDep := range rawDeps {
			printJSON(rawDep, "")

			if *resolve {
				fmt.Println("# resolves to:")
				resolvedDep, err := dep2.Resolve(rawDep, c, x)
				if err != nil {
					log.Fatal(err)
				}
				printJSON(resolvedDep, "  ")
			}
		}

		fmt.Println()
	}
}

func graph_(args []string) {
	fs := flag.NewFlagSet("graph", flag.ExitOnError)
	r := addRepositoryFlags(fs)
	jsonOutput := fs.Bool("json", false, "show JSON output")
	summary := fs.Bool("summary", true, "summarize output data")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: srcgraph graph [options] [unit...]

Analyze a repository's source code for definitions and references. If unit(s)
are specified, only source units with matching IDs will be graphed.

The options are:
`)
		fs.PrintDefaults()
		os.Exit(1)
	}
	fs.Parse(args)

	sourceUnitIDs := fs.Args()

	repoURI := repo.MakeURI(r.cloneURL)

	x := task2.DefaultContext

	c, err := scan.ReadDirConfigAndScan(r.rootDir, repoURI, x)
	if err != nil {
		log.Fatal(err)
	}

	for _, u := range c.SourceUnits {
		var match bool
		if len(sourceUnitIDs) == 0 {
			match = true
		} else {
			for _, unitID := range sourceUnitIDs {
				if u.ID() == unitID {
					match = true
					break
				}
			}
		}

		if !match {
			if *verbose {
				log.Printf("Skipping source unit %s", u.ID())
			}
			continue
		}

		log.Printf("## %s", u.ID())

		output, err := grapher2.Graph(r.rootDir, u, c, task2.DefaultContext)
		if err != nil {
			log.Fatal(err)
		}

		if *summary || *verbose {
			log.Printf("## %s output summary:", u.ID())
			log.Printf(" - %d symbols", len(output.Symbols))
			log.Printf(" - %d refs", len(output.Refs))
			log.Printf(" - %d docs", len(output.Docs))
		}

		if *jsonOutput {
			printJSON(output, "")
		}

		fmt.Println()
	}
}

type repository struct {
	cloneURL    string
	commitID    string
	vcsTypeName string
	rootDir     string
}

func (r *repository) outputFile() string {
	absRootDir, err := filepath.Abs(r.rootDir)
	if err != nil {
		log.Fatal(err)
	}
	return filepath.Join(*tmpDir, fmt.Sprintf("%s-%s.json", filepath.Base(absRootDir), r.commitID))
}

func detectRepository(dir string) (dr repository) {
	rootDirCmds := map[string]*exec.Cmd{
		"git": exec.Command("git", "rev-parse", "--show-toplevel"),
		"hg":  exec.Command("hg", "root"),
	}
	for tn, cmd := range rootDirCmds {
		cmd.Dir = dir
		out, err := cmd.Output()
		if err != nil && *verbose {
			log.Printf("warning: failed to find %s repository root dir in %s: %s", tn, dir, err)
			continue
		}
		if err == nil {
			dr.rootDir = strings.TrimSpace(string(out))
			dr.vcsTypeName = tn
			break
		}
	}

	if dr.rootDir == "" {
		if *verbose {
			log.Printf("warning: failed to detect repository root dir")
		}
		return
	}

	cloneURLCmd := map[string]*exec.Cmd{
		"git": exec.Command("git", "config", "remote.origin.url"),
		"hg":  exec.Command("hg", "paths", "default"),
	}[dr.vcsTypeName]

	vcsType := vcs.VCSByName[dr.vcsTypeName]
	repo, err := vcs.Open(vcsType, dr.rootDir)
	if err != nil {
		if *verbose {
			log.Printf("warning: failed to open repository at %s: %s", dr.rootDir, err)
		}
		return
	}

	dr.commitID, err = repo.CurrentCommitID()
	if err != nil {
		return
	}

	cloneURLCmd.Dir = dir
	cloneURL, err := cloneURLCmd.Output()
	if err != nil {
		return
	}
	dr.cloneURL = strings.TrimSpace(string(cloneURL))

	if dr.vcsTypeName == "git" {
		dr.cloneURL = strings.Replace(dr.cloneURL, "git@github.com:", "git://github.com/", 1)
	}

	return
}

func addRepositoryFlags(fs *flag.FlagSet) repository {
	dr := detectRepository(*dir)
	var r repository
	fs.StringVar(&r.cloneURL, "cloneurl", dr.cloneURL, "clone URL of repository")
	fs.StringVar(&r.commitID, "commit", dr.commitID, "commit ID of current working tree")
	fs.StringVar(&r.vcsTypeName, "vcs", dr.vcsTypeName, `VCS type ("git" or "hg")`)
	fs.StringVar(&r.rootDir, "root", dr.rootDir, `root directory of repository`)
	return r
}

func isDir(dir string) bool {
	di, err := os.Stat(dir)
	return err == nil && di.IsDir()
}

func mkTmpDir() {
	err := os.MkdirAll(*tmpDir, 0700)
	if err != nil {
		log.Fatal(err)
	}
}

func printJSON(v interface{}, prefix string) {
	data, err := json.MarshalIndent(v, prefix, "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
