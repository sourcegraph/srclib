package src

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"strings"
	"sync"

	"github.com/aybabtme/color/brush"
	"github.com/sqs/go-flags"
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

	_, err = c.AddCommand("add",
		"add a local toolchain",
		"Add a local directory as a toolchain in SRCLIBPATH.",
		&toolchainAddCmd,
	)
	if err != nil {
		log.Fatal(err)
	}

	_, err = c.AddCommand("install-std",
		"install standard toolchains",
		"Install standard toolchains (sourcegraph.com/sourcegraph/srclib-* toolchains).",
		&toolchainInstallStdCmd,
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
		_, err := toolchain.Get(string(tc), c.Update)
		if err != nil {
			return err
		}
	}
	return nil
}

type ToolchainAddCmd struct {
	Dir  string `long:"dir" description:"directory containing toolchain to add" value-name:"DIR"`
	Args struct {
		ToolchainPath string `name:"TOOLCHAIN" default:"." description:"toolchain path to use for toolchain directory"`
	} `positional-args:"yes" required:"yes"`
}

var toolchainAddCmd ToolchainAddCmd

func (c *ToolchainAddCmd) Execute(args []string) error {
	return toolchain.Add(c.Dir, c.Args.ToolchainPath)
}

type ToolchainInstallStdCmd struct {
	Skip []string `long:"skip" description:"skip installing matching toolchains (can be specified multiple times; e.g., --skip go --skip ruby)" value-name:"NAME"`
}

var toolchainInstallStdCmd ToolchainInstallStdCmd

func (c *ToolchainInstallStdCmd) Execute(args []string) error {
	fmt.Println(brush.Cyan("Installing/upgrading standard toolchains..."))
	fmt.Println()

	x := []struct {
		name string
		fn   func() error
	}{
		{"Go (sourcegraph.com/sourcegraph/srclib-go)", c.installGoToolchain},
		{"Python (sourcegraph.com/sourcegraph/srclib-python)", c.installPythonToolchain},
		{"Ruby (sourcegraph.com/sourcegraph/srclib-ruby)", c.installRubyToolchain},
		{"JavaScript (sourcegraph.com/sourcegraph/srclib-javascript)", c.installJavaScriptToolchain},
	}

OuterLoop:
	for _, x := range x {
		name := x.name
		for _, skip := range c.Skip {
			if strings.Contains(name, skip) {
				fmt.Println(brush.Yellow(fmt.Sprintf("Skipping installation of %s", name)))
				continue OuterLoop
			}
		}
		fmt.Println(brush.Cyan(name + " " + strings.Repeat("=", 78-len(name))).String())
		if err := x.fn(); err != nil {
			if err, ok := err.(skippedToolchain); ok {
				fmt.Println(brush.Yellow(err.Error()).String())
				fmt.Println()
				continue
			}
			return errors.New(brush.Red(fmt.Sprintf("failed to install/upgrade %s toolchain: %s", name, err)).String())
		}

		fmt.Println(brush.Green("OK! Installed/upgraded " + name + " toolchain").String())
		fmt.Println(brush.Cyan(strings.Repeat("=", 80)).String())
		fmt.Println()
	}

	return nil
}

func (c *ToolchainInstallStdCmd) installGoToolchain() error {
	const toolchain = "sourcegraph.com/sourcegraph/srclib-go"
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		return skippedToolchain{toolchain, "no GOPATH set (assuming Go is not installed and you don't want the Go toolchain)"}
	}

	srclibpathDir := filepath.Join(strings.Split(srclib.Path, ":")[0], toolchain) // toolchain dir under SRCLIBPATH

	if err := os.MkdirAll(filepath.Dir(srclibpathDir), 0700); err != nil {
		return err
	}

	if skipmsg, err := symlinkToGopath(toolchain); err != nil {
		return err
	} else if skipmsg != "" {
		return skippedToolchain{toolchain, skipmsg}
	}

	log.Println("Downloading or updating Go toolchain in", srclibpathDir)
	if err := execCmd("src", "toolchain", "get", "-u", toolchain); err != nil {
		return err
	}

	log.Println("Building Go toolchain program")
	if err := execCmd("make", "-C", srclibpathDir); err != nil {
		return err
	}

	return nil
}

func (c *ToolchainInstallStdCmd) installRubyToolchain() error {
	const toolchain = "sourcegraph.com/sourcegraph/srclib-ruby"

	srclibpathDir := filepath.Join(strings.Split(srclib.Path, ":")[0], toolchain) // toolchain dir under SRCLIBPATH

	if _, err := exec.LookPath("ruby"); isExecErrNotFound(err) {
		return skippedToolchain{toolchain, "no `ruby` in PATH (assuming you don't have Ruby installed and you don't want the Ruby toolchain)"}
	}
	if _, err := exec.LookPath("bundle"); isExecErrNotFound(err) {
		return fmt.Errorf("found `ruby` in PATH but did not find `bundle` in PATH; Ruby toolchain requires bundler (run `gem install bundler` to install it)")
	}

	log.Println("Downloading or updating Ruby toolchain in", srclibpathDir)
	if err := execCmd("src", "toolchain", "get", "-u", toolchain); err != nil {
		return err
	}

	log.Println("Installing deps for Ruby toolchain in", srclibpathDir)
	if err := execCmd("make", "-C", srclibpathDir); err != nil {
		return fmt.Errorf("%s\n\nTip: If you are using a version of Ruby other than 2.1.2 (the default for srclib), or if you are using your system Ruby, try using a Ruby version manager (such as https://rvm.io) to install a more standard Ruby, and try Ruby 2.1.2.\n\nIf you are still having problems, post an issue at https://github.com/sourcegraph/srclib-ruby/issues with the full log output and information about your OS and Ruby version.\n\nIf you don't care about Ruby, skip this installation by running `src toolchain install-std --skip ruby`.", err)
	}

	return nil
}

func (c *ToolchainInstallStdCmd) installJavaScriptToolchain() error {
	const toolchain = "sourcegraph.com/sourcegraph/srclib-javascript"

	srclibpathDir := filepath.Join(strings.Split(srclib.Path, ":")[0], toolchain) // toolchain dir under SRCLIBPATH

	if _, err := exec.LookPath("node"); isExecErrNotFound(err) {
		return skippedToolchain{toolchain, "no `node` in PATH (assuming you don't have Node.js installed and you don't want the JavaScript toolchain)"}
	}
	if _, err := exec.LookPath("npm"); isExecErrNotFound(err) {
		return fmt.Errorf("no `npm` in PATH; JavaScript toolchain requires npm")
	}

	log.Println("Downloading or updating JavaScript toolchain in", srclibpathDir)
	if err := execCmd("src", "toolchain", "get", "-u", toolchain); err != nil {
		return err
	}

	return nil
}

func (c *ToolchainInstallStdCmd) installPythonToolchain() error {
	const toolchain = "sourcegraph.com/sourcegraph/srclib-python"

	requiredCmds := map[string]string{
		"go":         "visit https://golang.org/doc/install",
		"python":     "visit https://www.python.org/downloads/",
		"pip":        "visit http://pip.readthedocs.org/en/latest/installing.html",
		"virtualenv": "run `[sudo] pip install virtualenv`",
	}
	for requiredCmd, instructions := range requiredCmds {
		if _, err := exec.LookPath(requiredCmd); isExecErrNotFound(err) {
			return skippedToolchain{toolchain, fmt.Sprintf("no `%s` found in PATH; to install, %s", requiredCmd, instructions)}
		}
	}

	srclibpathDir := filepath.Join(strings.Split(srclib.Path, ":")[0], toolchain) // toolchain dir under SRCLIBPATH
	log.Println("Downloading or updating Python toolchain in", srclibpathDir)
	if err := execCmd("src", "toolchain", "get", "-u", toolchain); err != nil {
		return err
	}

	// Add symlink to GOPATH so install succeeds (necessary as long as there's a Go dependency in this toolchain)
	if skipmsg, err := symlinkToGopath(toolchain); err != nil {
		return err
	} else if skipmsg != "" {
		return skippedToolchain{toolchain, skipmsg}
	}

	log.Println("Installing deps for Python toolchain in", srclibpathDir)
	if err := execCmd("make", "-C", srclibpathDir, "install"); err != nil {
		return err
	}

	return nil
}

type skippedToolchain struct {
	toolchain string
	why       string
}

func (e skippedToolchain) Error() string {
	return fmt.Sprintf("skipped %s: %s", e.toolchain, e.why)
}

func isExecErrNotFound(err error) bool {
	if e, ok := err.(*exec.Error); ok && e.Err == exec.ErrNotFound {
		return true
	}
	return false
}

func symlinkToGopath(toolchain string) (skip string, err error) {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		return "", fmt.Errorf("GOPATH not set")
	}

	srcDir := filepath.Join(strings.Split(gopath, ":")[0], "src")
	gopathDir := filepath.Join(srcDir, toolchain)
	srclibpathDir := filepath.Join(strings.Split(srclib.Path, ":")[0], toolchain)

	if fi, err := os.Lstat(gopathDir); os.IsNotExist(err) {
		log.Printf("mkdir -p %s", filepath.Dir(gopathDir))
		if err := os.MkdirAll(filepath.Dir(gopathDir), 0700); err != nil {
			return "", err
		}
		log.Printf("ln -s %s %s", srclibpathDir, gopathDir)
		if err := os.Symlink(srclibpathDir, gopathDir); err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	} else if fi.Mode()&os.ModeSymlink == 0 {
		return fmt.Sprintf("toolchain dir in GOPATH (%s) is not a symlink (assuming you intentionally cloned the toolchain repo to your GOPATH; not modifying it)", gopathDir), nil
	}

	log.Printf("Symlinked toolchain %s into your GOPATH at", toolchain, gopathDir)
	return "", nil
}
