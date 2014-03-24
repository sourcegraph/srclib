package srcgraph

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kr/text"
	"github.com/sourcegraph/go-vcs"
	"sourcegraph.com/sourcegraph/client"
	"sourcegraph.com/sourcegraph/config2"
	"sourcegraph.com/sourcegraph/repo"
	"sourcegraph.com/sourcegraph/srcgraph/build"
	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/dep2"
	"sourcegraph.com/sourcegraph/srcgraph/grapher2"
	"sourcegraph.com/sourcegraph/srcgraph/scan"
	"sourcegraph.com/sourcegraph/srcgraph/task2"
	"sourcegraph.com/sourcegraph/srcgraph/toolchain"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
	"sourcegraph.com/sourcegraph/srcgraph/util2/makefile"
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
		for _, c := range subcommands {
			fmt.Fprintf(os.Stderr, "    %-10s %s\n", c.name, c.description)
		}
		fmt.Fprintln(os.Stderr, `
Use "`+Name+` command -h" for more information about a command.

The options are:
`)
		flag.PrintDefaults()
		os.Exit(1)
	}
}

var verbose = flag.Bool("v", false, "show verbose output")
var dir = flag.String("dir", ".", "directory to work in")
var tmpDir = flag.String("tmpdir", build.WorkDir, "temporary directory to use")

var apiclient = client.NewClient(nil)

func Main() {
	flag.Parse()

	if flag.NArg() == 0 {
		flag.Usage()
	}
	log.SetFlags(0)
	log.SetPrefix("")
	defer task2.FlushAll()

	subcmd := flag.Arg(0)
	for _, c := range subcommands {
		if c.name == subcmd {
			c.run(flag.Args()[1:])
			return
		}
	}

	fmt.Fprintf(os.Stderr, Name+": unknown subcommand %q\n", subcmd)
	fmt.Fprintln(os.Stderr, `Run "`+Name+` -h" for usage.`)
	os.Exit(1)
}

type subcommand struct {
	name        string
	description string
	run         func(args []string)
}

var subcommands = []subcommand{
	{"build", "build a repository", build_},
	{"data", "list repository data", data},
	{"upload", "upload a previously generated build", upload},
	{"push", "update a repository and related information on Sourcegraph", push},
	{"scan", "scan a repository for source units", scan_},
	{"config", "validate and print a repository's configuration", config_},
	{"list-deps", "list a repository's raw (unresolved) dependencies", listDeps},
	{"resolve-deps", "resolve a repository's raw dependencies", resolveDeps},
	{"graph", "analyze a repository's source code for definitions and references", graph_},
	{"info", "show info about enabled capabilities", info},
}

func build_(args []string) {
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	repo := addRepositoryFlags(fs)
	dryRun := fs.Bool("n", false, "dry run (scans the repository and just prints out what analysis tasks would be performed)")
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

	rules, err := build.CreateMakefile(repo.rootDir, repo.cloneURL, repo.commitID, x)
	if err != nil {
		log.Fatalf("error creating Makefile: %s", err)
	}

	if *verbose || *dryRun {
		mf, err := makefile.Makefile(rules)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("# Makefile\n%s", mf)
	}
	if *dryRun {
		return
	}

	err = makefile.MakeRules(repo.rootDir, rules)
	if err != nil {
		log.Fatalf("make failed: %s", err)
	}

	if *verbose {
		if len(rules) > 0 {
			log.Printf("%d output files:", len(rules))
			for _, r := range rules {
				log.Printf(" - %s", r.Target().Name())
			}
		}
	}
}

func upload(args []string) {
	fs := flag.NewFlagSet("upload", flag.ExitOnError)
	r := addRepositoryFlags(fs)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: srcgraph upload [options]

Uploads build data for a repository to Sourcegraph.

The options are:
`)
		fs.PrintDefaults()
		os.Exit(1)
	}
	fs.Parse(args)

	if fs.NArg() != 0 {
		fs.Usage()
	}

	x := task2.NewRecordedContext()
	repoURI := repo.MakeURI(r.cloneURL)

	rules, err := build.CreateMakefile(r.rootDir, r.cloneURL, r.commitID, x)
	if err != nil {
		log.Fatalf("error creating Makefile: %s", err)
	}

	for _, rule := range rules {
		uploadFile(rule.Target(), repoURI, r.commitID)
	}
}

func uploadFile(target makefile.Target, repoURI repo.URI, commitID string) {
	fi, err := os.Stat(target.Name())
	if err != nil || !fi.Mode().IsRegular() {
		if *verbose {
			log.Printf("upload: skipping nonexistent file %s", target.Name())
		}
		return
	}

	kb := float64(fi.Size()) / 1024
	if *verbose {
		log.Printf("Uploading %s (%.1fkb)", target.Name(), kb)
	}

	f, err := os.Open(target.Name())
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	_, err = apiclient.BuildData.Upload(client.BuildDatumSpec{RepositorySpec: client.RepositorySpec{URI: string(repoURI)}, CommitID: commitID, Name: target.(build.Target).RelName()}, f)
	if err != nil {
		log.Fatal(err)
	}

	if *verbose {
		log.Printf("Uploaded %s (%.1fkb)", target.Name(), kb)
	}
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
		fmt.Printf("## %s\n", unit.MakeID(u))
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

func listDeps(args []string) {
	fs := flag.NewFlagSet("list-deps", flag.ExitOnError)
	resolve := fs.Bool("resolve", false, "resolve deps and print resolutions")
	jsonOutput := fs.Bool("json", false, "show JSON output")
	r := addRepositoryFlags(fs)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: srcgraph list-deps [options] [unit...]

Lists a repository's raw (unresolved) dependencies. If unit(s) are specified,
only source units with matching IDs will have their dependencies listed.

The options are:
`)
		fs.PrintDefaults()
		os.Exit(1)
	}
	fs.Parse(args)
	sourceUnitSpecs := fs.Args()
	repoURI := repo.MakeURI(r.cloneURL)

	x := task2.DefaultContext
	c, err := scan.ReadDirConfigAndScan(r.rootDir, repoURI, x)
	if err != nil {
		log.Fatal(err)
	}

	allRawDeps := []*dep2.RawDependency{}
	for _, u := range c.SourceUnits {
		if !sourceUnitMatchesArgs(sourceUnitSpecs, u) {
			continue
		}

		rawDeps, err := dep2.List(r.rootDir, u, c, x)
		if err != nil {
			log.Fatal(err)
		}

		if *verbose {
			log.Printf("## %s", unit.MakeID(u))
		}

		allRawDeps = append(allRawDeps, rawDeps...)

		for _, rawDep := range rawDeps {
			if *verbose {
				log.Printf("%+v", rawDep)
			}

			if *resolve {
				log.Printf("# resolves to:")
				resolvedDep, err := dep2.Resolve(rawDep, c, x)
				if err != nil {
					log.Fatal(err)
				}
				log.Printf("%+v", resolvedDep)
			}
		}
	}

	if *jsonOutput {
		printJSON(allRawDeps, "")
	}
}

func resolveDeps(args []string) {
	fs := flag.NewFlagSet("resolve-deps", flag.ExitOnError)
	r := addRepositoryFlags(fs)
	jsonOutput := fs.Bool("json", false, "show JSON output")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: srcgraph resolve-deps [options] [raw_dep_file.json...]

Resolves a repository's raw dependencies. If no files are specified, input is
read from stdin.

The options are:
`)
		fs.PrintDefaults()
		os.Exit(1)
	}
	fs.Parse(args)
	repoURI := repo.MakeURI(r.cloneURL)
	inputs := make(map[string]io.Reader)
	if fs.NArg() == 0 {
		inputs["<stdin>"] = os.Stdin
	} else {
		for _, name := range fs.Args() {
			f, err := os.Open(name)
			if err != nil {
				log.Fatal(err)
			}
			defer f.Close()
			inputs[name] = f
		}
	}

	x := task2.DefaultContext
	c, err := scan.ReadDirConfigAndScan(r.rootDir, repoURI, x)
	if err != nil {
		log.Fatal(err)
	}

	var allRawDeps []*dep2.RawDependency
	for name, input := range inputs {
		var rawDeps []*dep2.RawDependency
		err := json.NewDecoder(input).Decode(&rawDeps)
		if err != nil {
			log.Fatalf("%s: %s", name, err)
		}

		allRawDeps = append(allRawDeps, rawDeps...)
	}

	resolvedDeps, err := dep2.ResolveAll(allRawDeps, c, x)
	if err != nil {
		log.Fatal(err)
	}
	if resolvedDeps == nil {
		resolvedDeps = []*dep2.ResolvedDep{}
	}

	if *jsonOutput {
		printJSON(resolvedDeps, "")
	}
}

func data(args []string) {
	fs := flag.NewFlagSet("data", flag.ExitOnError)
	r := detectRepository(*dir)
	repoURI := fs.String("repo", string(repo.MakeURI(r.cloneURL)), "repository URI (ex: github.com/alice/foo)")
	commitID := fs.String("commit", r.commitID, "commit ID (optional)")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: srcgraph data [options]

Lists available repository data.

The options are:
`)
		fs.PrintDefaults()
		os.Exit(1)
	}
	fs.Parse(args)

	if fs.NArg() != 0 {
		fs.Usage()
	}

	var opt *client.BuildDataListOptions
	if *commitID != "" {
		opt = &client.BuildDataListOptions{CommitID: *commitID}
	}
	data, _, err := apiclient.BuildData.List(client.RepositorySpec{URI: *repoURI}, opt)
	if err != nil {
		log.Fatal(err)
	}

	printJSON(data, "")
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
	sourceUnitSpecs := fs.Args()
	repoURI := repo.MakeURI(r.cloneURL)

	x := task2.DefaultContext
	c, err := scan.ReadDirConfigAndScan(r.rootDir, repoURI, x)
	if err != nil {
		log.Fatal(err)
	}

	for _, u := range c.SourceUnits {
		if !sourceUnitMatchesArgs(sourceUnitSpecs, u) {
			continue
		}

		log.Printf("## %s", unit.MakeID(u))

		output, err := grapher2.Graph(r.rootDir, u, c, task2.DefaultContext)
		if err != nil {
			log.Fatal(err)
		}

		if *summary || *verbose {
			log.Printf("## %s output summary:", unit.MakeID(u))
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

func info(args []string) {
	log.Printf("Toolchains (%d)", len(toolchain.Toolchains))
	for tcName, _ := range toolchain.Toolchains {
		log.Printf(" - %s", tcName)
	}
	log.Println()

	log.Printf("Config global sections (%d)", len(config.Globals))
	for name, typ := range config.Globals {
		log.Printf(" - %s (type %T)", name, typ)
	}
	log.Println()

	log.Printf("Source units (%d)", len(unit.Types))
	for name, typ := range unit.Types {
		log.Printf(" - %s (type %T)", name, typ)
	}
	log.Println()

	log.Printf("Scanners (%d)", len(scan.Scanners))
	for name, _ := range scan.Scanners {
		log.Printf(" - %s", name)
	}
	log.Println()

	log.Printf("Graphers (%d)", len(grapher2.Graphers))
	for typ, _ := range grapher2.Graphers {
		log.Printf(" - %s source units", unit.TypeNames[typ])
	}
	log.Println()

	log.Printf("Dependency raw listers (%d)", len(dep2.Listers))
	for typ, _ := range dep2.Listers {
		log.Printf(" - %s source units", unit.TypeNames[typ])
	}
	log.Println()

	log.Printf("Dependency resolvers (%d)", len(dep2.Resolvers))
	for typ, _ := range dep2.Resolvers {
		log.Printf(" - %q raw dependencies", typ)
	}
	log.Println()

	log.Printf("Build rule makers (%d)", len(build.RuleMakers))
	for name, _ := range build.RuleMakers {
		log.Printf(" - %s", name)
	}
	log.Println()

	log.Printf("------------------")
	log.Println()
	log.Printf("System information:")
	log.Printf(" - make version: %s", firstLine(cmdOutput("make", "--version")))
	log.Printf(" - docker version:\n%s", text.Indent(cmdOutput("docker", "version"), "         "))
}

func firstLine(s string) string {
	i := strings.Index(s, "\n")
	if i == -1 {
		return s
	}
	return s[:i]
}

func cmdOutput(c ...string) string {
	cmd := exec.Command(c[0], c[1:]...)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		log.Fatalf("%v: %s", c, err)
	}
	return strings.TrimSpace(string(out))
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
	if !isDir(dir) {
		log.Fatal("dir does not exist: ", dir)
	}

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

func isFile(file string) bool {
	fi, err := os.Stat(file)
	return err == nil && fi.Mode().IsRegular()
}

func mkTmpDir() {
	err := os.MkdirAll(*tmpDir, 0700)
	if err != nil {
		log.Fatal(err)
	}
}

func sourceUnitMatchesArgs(specified []string, u unit.SourceUnit) bool {
	var match bool
	if len(specified) == 0 {
		match = true
	} else {
		for _, unitSpec := range specified {
			if string(unit.MakeID(u)) == unitSpec || u.Name() == unitSpec {
				match = true
				break
			}
		}
	}

	if !match {
		if *verbose {
			log.Printf("Skipping source unit %s", unit.MakeID(u))
		}
	}

	return match
}

func printJSON(v interface{}, prefix string) {
	data, err := json.MarshalIndent(v, prefix, "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
