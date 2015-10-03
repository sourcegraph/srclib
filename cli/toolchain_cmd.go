package cli

import (
	"errors"
	"fmt"
	"go/build"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"time"

	"strings"
	"sync"

	ask "github.com/GeertJohan/go.ask"
	"github.com/aybabtme/color/brush"
	flags "sourcegraph.com/sourcegraph/go-flags"
	"sourcegraph.com/sourcegraph/srclib"
	"sourcegraph.com/sourcegraph/srclib/toolchain"
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
	c.Aliases = []string{"tc"}

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

	_, err = c.AddCommand("bundle",
		"bundle a toolchain",
		"The bundle subcommand builds and archives toolchain bundles (.tar.gz files, one per toolchain variant). Bundles contain prebuilt toolchains and allow people to use srclib toolchains without needing to compile them on their own system.",
		&toolchainBundleCmd,
	)
	if err != nil {
		log.Fatal(err)
	}

	_, err = c.AddCommand("unbundle",
		"unbundle a toolchain",
		"The unbundle subcommand unarchives a toolchain bundle (previously created with the 'bundle' subcommand). It allows people to download and use prebuilt toolchains without needing to compile them on their system.",
		&toolchainUnbundleCmd,
	)
	if err != nil {
		log.Fatal(err)
	}

	_, err = c.AddCommand("add",
		"add a local toolchain",
		"Add a local directory as a toolchain in SRCLIBPATH.",
		&toolchainAddCmd,
	)
	if err != nil {
		log.Fatal(err)
	}

	_, err = c.AddCommand("install",
		"install toolchains",
		"Download and install toolchains",
		&toolchainInstallCmd,
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
	ExeMethods string `short:"m" long:"methods" default:"program,docker" description:"toolchain execution methods" value-name:"METHODS"`
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
	Op             string `short:"p" long:"op" description:"only list tools that perform these operations only" value-name:"OP"`
	SourceUnitType string `short:"u" long:"source-unit-type" description:"only list tools that operate on this source unit type" value-name:"TYPE"`
	Args           struct {
		Toolchains []ToolchainPath `name:"TOOLCHAINS" description:"only list tools in these toolchains"`
	} `positional-args:"yes" required:"yes"`
}

var toolchainListToolsCmd ToolchainListToolsCmd

func (c *ToolchainListToolsCmd) Execute(args []string) error {
	tcs, err := toolchain.List()
	if err != nil {
		log.Fatal(err)
	}

	fmtStr := "%-40s  %-18s  %-15s  %-25s\n"
	fmt.Printf(fmtStr, "TOOLCHAIN", "TOOL", "OP", "SOURCE UNIT TYPES")
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

		cfg, err := tc.ReadConfig()
		if err != nil {
			log.Fatal(err)
		}
		for _, t := range cfg.Tools {
			if c.Op != "" && c.Op != t.Op {
				continue
			}
			if c.SourceUnitType != "" {
				found := false
				for _, u := range t.SourceUnitTypes {
					if c.SourceUnitType == u {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}

			fmt.Printf(fmtStr, tc.Path, t.Subcmd, t.Op, strings.Join(t.SourceUnitTypes, " "))
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
	var wg sync.WaitGroup
	for _, tc := range c.Args.Toolchains {
		tc, err := toolchain.Open(string(tc), toolchain.AsDockerContainer)
		if err != nil {
			log.Fatal(err)
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := tc.Build(); err != nil {
				log.Fatal(err)
			}
		}()
	}
	wg.Wait()
	return nil
}

type ToolchainGetCmd struct {
	Update bool `short:"u" long:"update" description:"use the network to update the toolchain"`
	Args   struct {
		Toolchains []ToolchainPath `name:"TOOLCHAINS" description:"toolchain paths of toolchains to get"`
	} `positional-args:"yes" required:"yes"`
}

var toolchainGetCmd ToolchainGetCmd

func (c *ToolchainGetCmd) Execute(args []string) error {
	for _, tc := range c.Args.Toolchains {
		if GlobalOpt.Verbose {
			fmt.Println(tc)
		}
		_, err := toolchain.CloneOrUpdate(string(tc), c.Update)
		if err != nil {
			return err
		}
	}
	return nil
}

type ToolchainBundleCmd struct {
	Variant string `long:"variant" description:"only produce a bundle for the given variant (default is all variants)"`
	DryRun  bool   `short:"n" long:"dry-run" description:"don't do anything, but print what would be done"`

	Args struct {
		Toolchain ToolchainPath `name:"TOOLCHAIN" description:"toolchain to bundle" required:"yes"`
		Dir       string        `name:"TOOLCHAIN-DIR" description:"dir containing toolchain files (default: look up TOOLCHAIN in SRCLIBPATH)"`
	} `positional-args:"yes"`
}

var toolchainBundleCmd ToolchainBundleCmd

func (c *ToolchainBundleCmd) Execute(args []string) error {
	log.Printf("Bundling toolchain %s...", c.Args.Toolchain)

	tmpDir, err := ioutil.TempDir("", path.Base(string(c.Args.Toolchain))+"toolchain-bundle")
	if err != nil {
		return err
	}
	log.Printf(" - output dir: %s", tmpDir)

	var variants []toolchain.Variant
	if c.Variant != "" {
		variants = append(variants, toolchain.ParseVariant(c.Variant))
	}

	if c.Args.Dir == "" {
		info, err := toolchain.Lookup(string(c.Args.Toolchain))
		if err != nil {
			return err
		}
		c.Args.Dir = info.Dir
	}

	bundles, err := toolchain.Bundle(c.Args.Dir, tmpDir, variants, c.DryRun, GlobalOpt.Verbose)
	if err != nil {
		return err
	}

	log.Println()
	log.Println("Bundles ready:", tmpDir)
	for _, b := range bundles {
		log.Println("   ", b)
	}

	return nil
}

type ToolchainUnbundleCmd struct {
	Args struct {
		Toolchain  string `name:"TOOLCHAIN" description:"toolchain path to unbundle to"`
		BundleFile string `name:"BUNDLE-FILE" description:"bundle file containing toolchain dir contents (.tar.gz, .tar, etc.)"`
	} `positional-args:"yes" required:"yes"`
}

var toolchainUnbundleCmd ToolchainUnbundleCmd

func (c *ToolchainUnbundleCmd) Execute(args []string) error {
	if GlobalOpt.Verbose {
		log.Printf("Unarchiving from bundle file %s to toolchain %s", c.Args.BundleFile, c.Args.Toolchain)
	}

	f, err := os.Open(c.Args.BundleFile)
	if err != nil {
		return err
	}
	defer f.Close()
	return toolchain.Unbundle(c.Args.Toolchain, c.Args.BundleFile, f)
}

type ToolchainAddCmd struct {
	Dir   string `long:"dir" description:"directory containing toolchain to add" value-name:"DIR"`
	Force bool   `short:"f" long:"force" description:"(dangerous) force add, overwrite existing toolchain"`
	Args  struct {
		ToolchainPath string `name:"TOOLCHAIN" default:"." description:"toolchain path to use for toolchain directory"`
	} `positional-args:"yes" required:"yes"`
}

var toolchainAddCmd ToolchainAddCmd

func (c *ToolchainAddCmd) Execute(args []string) error {
	return toolchain.Add(c.Dir, c.Args.ToolchainPath, &toolchain.AddOpt{Force: c.Force})
}

type toolchainInstaller struct {
	name string
	fn   func() error
}

type toolchainMap map[string]toolchainInstaller

var stdToolchains = toolchainMap{
	"go":         toolchainInstaller{"Go (sourcegraph.com/sourcegraph/srclib-go)", installGoToolchain},
	"python":     toolchainInstaller{"Python (sourcegraph.com/sourcegraph/srclib-python)", installPythonToolchain},
	"ruby":       toolchainInstaller{"Ruby (sourcegraph.com/sourcegraph/srclib-ruby)", installRubyToolchain},
	"javascript": toolchainInstaller{"JavaScript (sourcegraph.com/sourcegraph/srclib-javascript)", installJavaScriptToolchain},
	"java":       toolchainInstaller{"Java (sourcegraph.com/sourcegraph/srclib-java)", installJavaToolchain},
}

func (m toolchainMap) listKeys() string {
	var langs string
	for i, _ := range m {
		langs += i + ", "
	}
	// Remove the last comma from langs before returning it.
	return strings.TrimSuffix(langs, ", ")
}

type ToolchainInstallCmd struct {
	// Args are not required so we can print out a more detailed
	// error message inside (*ToolchainInstallCmd).Execute.
	Args struct {
		Languages []string `value-name:"LANG" description:"language toolchains to install"`
	} `positional-args:"yes"`
}

var toolchainInstallCmd ToolchainInstallCmd

func (c *ToolchainInstallCmd) Execute(args []string) error {
	if len(c.Args.Languages) == 0 {
		return errors.New(brush.Red(fmt.Sprintf("No languages specified. Standard languages include: %s", stdToolchains.listKeys())).String())
	}
	var is []toolchainInstaller
	for _, l := range c.Args.Languages {
		i, ok := stdToolchains[l]
		if !ok {
			return errors.New(brush.Red(fmt.Sprintf("Language %s unrecognized. Standard languages include: %s", l, stdToolchains.listKeys())).String())
		}
		is = append(is, i)
	}
	return installToolchains(is)
}

func installToolchains(langs []toolchainInstaller) error {
	for _, l := range langs {
		fmt.Println(brush.Cyan(l.name + " " + strings.Repeat("=", 78-len(l.name))).String())
		if err := l.fn(); err != nil {
			return fmt.Errorf("%s\n", brush.Red(fmt.Sprintf("failed to install/upgrade %s toolchain: %s", l.name, err)))
		}

		fmt.Println(brush.Green("OK! Installed/upgraded " + l.name + " toolchain").String())
		fmt.Println(brush.Cyan(strings.Repeat("=", 80)).String())
		fmt.Println()
	}
	return nil
}

const goInstallErrorMessage = `
Error: %v

Refusing to install Go toolchain because Go installation has failed.

-> Install Go manually if possible.
-> Or contact support@sourcegraph.com (with the above output) for help.
`

// installGo tries to install Go using the system package manager (apt-get or
// brew), it returns an error if installation was attempted but failed, and
// returns nil if installation was not attempted (e.g. no supported package
// manager was found).
func installGo() error {
	cmds := func(cmds ...[]string) error {
		for _, args := range cmds {
			fmt.Println(strings.Join(args, " "))
			cmd := exec.Command(args[0], args[1:]...)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return err
			}
		}
		return nil
	}

	// apt-get installation
	if _, err := exec.LookPath("apt-get"); err == nil {
		if !ask.MustAsk("Install Go 1.5 via apt-get everlast PPA?") {
			return nil
		}

		err := cmds(
			// install add-apt-repository, which is needed on Ubuntu.
			[]string{"sudo", "apt-get", "install", "-y", "software-properties-common"},
			[]string{"sudo", "add-apt-repository", "-y", "ppa:evarlast/golang1.5"},
			[]string{"sudo", "apt-get", "-y", "update"},
			[]string{"sudo", "apt-get", "install", "-y", "golang-go-linux-" + build.Default.GOARCH},
		)
		if err != nil {
			return fmt.Errorf(goInstallErrorMessage, err)
		}
	}

	// brew installation
	if _, err := exec.LookPath("brew"); err == nil {
		if !ask.MustAsk("Install Go via homebrew?") {
			return nil
		}
		if err := cmds([]string{"brew", "install", "go"}); err != nil {
			return fmt.Errorf(goInstallErrorMessage, err)
		}
	}
	return nil
}

func installGoToolchain() error {
	const toolchain = "sourcegraph.com/sourcegraph/srclib-go"

	// Identify if Go is installed already or not.
	if _, err := exec.LookPath("go"); err != nil {
		if err := installGo(); err != nil {
			return err
		}
		return errors.New(`
Refusing to install Go toolchain because Go is not installed or is not on the
system path.

-> Please install the latest version of Go (https://golang.org/doc/install) and
run this command again.`)
	}

	if os.Getenv("GOPATH") == "" {
		os.Setenv("GOPATH", path.Join(os.Getenv("HOME"), ".srclib-gopath"))
	}
	// Add symlink to GOPATH so install succeeds (necessary as long as there's a Go dependency in this toolchain)
	if err := symlinkToGopath(toolchain); err != nil {
		return err
	}

	srclibpathDir := filepath.Join(strings.Split(srclib.Path, ":")[0], toolchain) // toolchain dir under SRCLIBPATH
	if err := os.MkdirAll(filepath.Dir(srclibpathDir), 0700); err != nil {
		return err
	}

	log.Println("Downloading or updating Go toolchain in", srclibpathDir)
	if err := execSrcCmd("toolchain", "get", "-u", toolchain); err != nil {
		return err
	}

	log.Println("Building Go toolchain program")
	if err := execCmd("make", "-C", srclibpathDir); err != nil {
		return err
	}

	return nil
}

func installRubyToolchain() error {
	const toolchain = "sourcegraph.com/sourcegraph/srclib-ruby"

	srclibpathDir := filepath.Join(strings.Split(srclib.Path, ":")[0], toolchain) // toolchain dir under SRCLIBPATH

	if _, err := exec.LookPath("ruby"); isExecErrNotFound(err) {
		return errors.New("no `ruby` in PATH (do you have Ruby installed properly?)")
	}
	if _, err := exec.LookPath("bundle"); isExecErrNotFound(err) {
		return fmt.Errorf("found `ruby` in PATH but did not find `bundle` in PATH; Ruby toolchain requires bundler (run `gem install bundler` to install it)")
	}

	log.Println("Downloading or updating Ruby toolchain in", srclibpathDir)
	if err := execSrcCmd("toolchain", "get", "-u", toolchain); err != nil {
		return err
	}

	log.Println("Installing deps for Ruby toolchain in", srclibpathDir)
	if err := execCmd("make", "-C", srclibpathDir); err != nil {
		return fmt.Errorf("%s\n\nTip: If you are using a version of Ruby other than 2.1.2 (the default for srclib), or if you are using your system Ruby, try using a Ruby version manager (such as https://rvm.io) to install a more standard Ruby, and try Ruby 2.1.2.\n\nIf you are still having problems, post an issue at https://github.com/sourcegraph/srclib-ruby/issues with the full log output and information about your OS and Ruby version.\n\n`.", err)
	}

	return nil
}

func installJavaScriptToolchain() error {
	const toolchain = "sourcegraph.com/sourcegraph/srclib-javascript"

	srclibpathDir := filepath.Join(strings.Split(srclib.Path, ":")[0], toolchain) // toolchain dir under SRCLIBPATH

	if _, err := exec.LookPath("node"); isExecErrNotFound(err) {
		return errors.New("no `node` in PATH (do you have Node.js installed properly?)")
	}
	if _, err := exec.LookPath("npm"); isExecErrNotFound(err) {
		return fmt.Errorf("no `npm` in PATH; JavaScript toolchain requires npm")
	}

	log.Println("Downloading or updating JavaScript toolchain in", srclibpathDir)
	if err := execSrcCmd("toolchain", "get", "-u", toolchain); err != nil {
		return err
	}

	return nil
}

func installPythonToolchain() error {
	const toolchain = "sourcegraph.com/sourcegraph/srclib-python"

	requiredCmds := map[string]string{
		"go":         "visit https://golang.org/doc/install",
		"python":     "visit https://www.python.org/downloads/",
		"pip":        "visit http://pip.readthedocs.org/en/latest/installing.html",
		"virtualenv": "run `[sudo] pip install virtualenv`",
	}
	for requiredCmd, instructions := range requiredCmds {
		if _, err := exec.LookPath(requiredCmd); isExecErrNotFound(err) {
			return fmt.Errorf("no `%s` found in PATH; to install, %s", requiredCmd, instructions)
		}
	}

	srclibpathDir := filepath.Join(strings.Split(srclib.Path, ":")[0], toolchain) // toolchain dir under SRCLIBPATH
	log.Println("Downloading or updating Python toolchain in", srclibpathDir)
	if err := execSrcCmd("toolchain", "get", "-u", toolchain); err != nil {
		return err
	}

	// Add symlink to GOPATH so install succeeds (necessary as long as there's a Go dependency in this toolchain)
	if err := symlinkToGopath(toolchain); err != nil {
		return err
	}

	log.Println("Installing deps for Python toolchain in", srclibpathDir)
	if err := execCmd("make", "-C", srclibpathDir, "install"); err != nil {
		return err
	}

	return nil
}

func installJavaToolchain() error {
	return installToolchainFromBundle("Java", "sourcegraph.com/sourcegraph/srclib-java", "https://srclib-support.s3-us-west-2.amazonaws.com/srclib-java/srclib-java__bundle__jdk-1.8.tar.gz")
}

func installToolchainFromBundle(name, toolchainPath, bundleURL string) (err error) {
	tmpDir, err := ioutil.TempDir("", "srclib-toolchain-bundle")
	if err != nil {
		return err
	}

	url, err := url.Parse(bundleURL)
	if err != nil {
		return err
	}

	log.Printf("Installing %s toolchain from %s", name, bundleURL)
	outputFile := filepath.Join(tmpDir, filepath.Base(url.Path))
	f, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer func() {
		if err2 := f.Close(); err2 != nil && err == nil {
			err = err2
		}
	}()

	resp, err := http.Get(bundleURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: unexpected HTTP status %d (%s)", bundleURL, resp.StatusCode, resp.Status)
	}

	done := make(chan struct{})
	go func() {
		for {
			t := time.After(time.Second * 2)
			select {
			case <-t:
				fi, err := os.Stat(outputFile)
				if err == nil {
					fmt.Printf("\rDownload %.1f%% complete (%.1f MB / %.1f MB)",
						float64(fi.Size())/float64(resp.ContentLength)*100,
						float64(fi.Size())/1024/1024,
						float64(resp.ContentLength)/1024/1024,
					)
				}
			case <-done:
				return
			}
		}
	}()

	log.Printf("Downloading %s (%.1f MB)", bundleURL, float64(resp.ContentLength)/1024/1024)
	_, err = io.Copy(f, resp.Body)
	done <- struct{}{}
	fmt.Println() // for the "\r" line in the progress indicator
	if err != nil {
		return err
	}

	log.Printf("Finished downloading")

	log.Printf("Unarchiving %s toolchain bundle at %s", name, outputFile)
	var unbundleCmd ToolchainUnbundleCmd
	unbundleCmd.Args.BundleFile = outputFile
	unbundleCmd.Args.Toolchain = toolchainPath
	if err := unbundleCmd.Execute(nil); err != nil {
		return err
	}

	return nil
}

func isExecErrNotFound(err error) bool {
	if e, ok := err.(*exec.Error); ok && e.Err == exec.ErrNotFound {
		return true
	}
	return false
}

func symlinkToGopath(toolchain string) error {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		return fmt.Errorf("GOPATH not set")
	}

	srcDir := filepath.Join(strings.Split(gopath, ":")[0], "src")
	gopathDir := filepath.Join(srcDir, toolchain)
	srclibpathDir := filepath.Join(strings.Split(srclib.Path, ":")[0], toolchain)

	if fi, err := os.Lstat(gopathDir); os.IsNotExist(err) {
		log.Printf("mkdir -p %s", filepath.Dir(gopathDir))
		if err := os.MkdirAll(filepath.Dir(gopathDir), 0700); err != nil {
			return err
		}
		log.Printf("ln -s %s %s", srclibpathDir, gopathDir)
		if err := os.Symlink(srclibpathDir, gopathDir); err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else if fi.Mode()&os.ModeSymlink == 0 {
		// toolchain dir in GOPATH is not a symlink, so assume they
		// intentionally cloned the toolchain repo into their GOPATH.
		return nil
	}

	log.Printf("Symlinked toolchain %s into your GOPATH at %s", toolchain, gopathDir)
	return nil
}
