package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/build"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"code.google.com/p/rog-go/parallel"
	"github.com/kr/text"

	"sort"
	"strings"

	"sourcegraph.com/sourcegraph/graph"
	"sourcegraph.com/sourcegraph/repo"
	"sourcegraph.com/sourcegraph/srcgraph/dep2"
	"sourcegraph.com/sourcegraph/srcgraph/grapher2"
	"sourcegraph.com/sourcegraph/srcgraph/scan"
	"sourcegraph.com/sourcegraph/srcgraph/task2"
	"sourcegraph.com/sourcegraph/srcgraph/toolchain"
	_ "sourcegraph.com/sourcegraph/srcgraph/toolchain/all_toolchains"
)

var updateExp = flag.Bool("exp", false, "update expected output files")
var verbose = flag.Bool("v", false, "verbose")
var dir = flag.String("dir", defaultBase("sourcegraph.com/sourcegraph/srcgraph/toolchain"), "toolchain dir")
var dirFilter = flag.String("q", "", "only test dirs containing this substring")

var allToolchainNames []string

func init() {
	for name, _ := range toolchain.Toolchains {
		allToolchainNames = append(allToolchainNames, name)
	}
	sort.Strings(allToolchainNames)
}

var exitCode int

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Tests the specified toolchain.\n\n")
		fmt.Fprintf(os.Stderr, "usage: sg-toolchain-test [OPTS] [TOOLCHAIN]...\n\n")
		fmt.Fprintf(os.Stderr, "with:\n")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintf(os.Stderr, "  TOOLCHAIN    name of toolchain (available: %v)\n", allToolchainNames)
		fmt.Fprintln(os.Stderr)
		fmt.Fprintf(os.Stderr, "and where OPTS is any of:\n")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr)
		fmt.Fprintf(os.Stderr, "For more about specifying packages, see 'go help packages'.\n")
		os.Exit(1)
	}
	flag.Parse()

	log.SetFlags(0)

	// Check that all specified toolchain names refer to registered toolchains.
	tcNames := flag.Args()
	for _, tcName := range tcNames {
		if _, registered := toolchain.Toolchains[tcName]; !registered {
			log.Fatalf("Toolchain %q is not registered. Registered toolchains are: %v.", tcName, allToolchainNames)
		}
	}

	errs := make(map[string]error)
	for _, tcName := range tcNames {
		errs[tcName] = testToolchain(tcName, toolchain.Toolchains[tcName])
	}

	for tcName, err := range errs {
		if err != nil {
			if errs, ok := err.(parallel.Errors); ok {
				log.Printf("ERRORS %s", tcName)
				for _, err := range errs {
					log.Printf("\t%s", err)
				}
			} else {
				log.Printf("ERROR %s: %s", tcName, err)
			}
		}
	}

	if *updateExp {
		log.Fatal("Updated expected output files. Now run without -exp to run tests.")
	}

	os.Exit(exitCode)
}

type testConfig struct {
	Dirs map[string]*testDirConfig
}

type testDirConfig struct {
	Scan            bool
	RawDependencies bool
	Graph           bool
}

func testToolchain(tcName string, tc toolchain.Toolchain) (err error) {
	if *verbose {
		log.Printf("RUN TOOLCHAIN %s", tcName)
		log.SetPrefix("\t")
	}

	defer func() {
		log.SetPrefix("")
		if err == nil {
			log.Printf("PASS %s", tcName)
		} else {
			exitCode = 1
			log.Printf("FAIL %s", tcName)
		}
	}()

	tcDir := filepath.Join(*dir, tcName)
	if !isDir(tcDir) {
		log.Printf("Toolchain dir %s does not exist.", tcDir)
		return os.ErrNotExist
	}

	testdataDir := filepath.Join(tcDir, "testdata")
	if !isDir(testdataDir) {
		log.Printf("Toolchain testdata dir %s does not exist.", testdataDir)
		return os.ErrNotExist
	}

	configFile := filepath.Join(testdataDir, "test.json")
	if !isFile(configFile) {
		log.Printf("Toolchain testdata dir %s is missing test.json file.", testdataDir)
		return os.ErrNotExist
	}

	var config testConfig
	f, err := os.Open(configFile)
	if err != nil {
		log.Printf("Opening %s failed: %s.", configFile, err)
		return err
	}
	defer f.Close()
	err = json.NewDecoder(f).Decode(&config)
	if err != nil {
		log.Printf("Decoding JSON in %s failed: %s.", configFile, err)
		return err
	}

	dirs := make(map[string]*testDirConfig)
	for glob, dirConfig := range config.Dirs {
		glob = filepath.Join(testdataDir, glob)
		matches, err := filepath.Glob(glob)
		if err != nil {
			log.Printf("Error expanding glob %q: %s.", glob, err)
			return err
		}

		for _, dir := range matches {
			if !isDir(dir) {
				log.Printf("Skipping non-directory %s (specified in test.json Dirs).", dir)
				continue
			}
			dir, _ = filepath.Rel(testdataDir, dir)
			if *dirFilter != "" && !strings.Contains(dir, *dirFilter) {
				if *verbose {
					log.Printf("Skipping unmatched directory %s.", dir)
				}
				continue
			}
			dirs[dir] = dirConfig
		}
	}

	par := parallel.NewRun(runtime.GOMAXPROCS(0))
	for dir_, dirConfig_ := range dirs {
		dir, dirConfig := dir_, dirConfig_
		par.Do(func() error {
			return testToolchainDir(tcName, tc, testdataDir, dir, dirConfig)
		})
	}

	return par.Wait()
}

func testToolchainDir(tcName string, tc toolchain.Toolchain, testdataDir, dir string, config *testDirConfig) (err error) {
	label := fmt.Sprintf("%s %s", tcName, dir)
	if *verbose {
		log.Printf("=== RUN %s", label)
		defer func() {
			if err == nil {
				log.Printf("--- PASS %s", label)
			} else {
				exitCode = 1
				log.Printf("--- FAIL %s\n%s", text.Indent(err.Error(), "\t"))
				err = fmt.Errorf("%s: %s", dir, err)
			}
		}()
	}

	x := task2.DefaultContext
	if !*verbose {
		x.Stdout, x.Stderr = ioutil.Discard, ioutil.Discard
		x.Log = log.New(ioutil.Discard, "", 0)
	}

	absDir := filepath.Join(testdataDir, dir)

	repoURI := repo.URI(fmt.Sprintf("example.com/%s/%s", tcName, dir))
	c, err := scan.ReadDirConfigAndScan(absDir, repoURI, x)
	if err != nil {
		log.Printf("ReadDirConfigAndScan in dir %s failed: %s.", absDir, err)
		return err
	}

	if config.Scan {
		err = output(filepath.Join(testdataDir, dir, "scan"), c)
		if err != nil {
			return err
		}
	}

	if config.RawDependencies {
		for _, u := range c.SourceUnits {
			rawDeps, err := dep2.List(absDir, u, c, x)
			if err != nil {
				log.Printf("List in dir %s failed: %s.", dir, err)
				return err
			}
			err = output(filepath.Join(testdataDir, dir, u.RootDir(), "raw_deps."+tcName), rawDeps)
			if err != nil {
				return err
			}
		}
	}

	if config.Graph {
		for _, u := range c.SourceUnits {
			graphOutput, err := grapher2.Graph(absDir, u, c, x)
			if err != nil {
				log.Printf("List in dir %s failed: %s.", dir, err)
				return err
			}

			sort.Sort(graph.Symbols(graphOutput.Symbols))
			sort.Sort(graph.Refs(graphOutput.Refs))

			err = output(filepath.Join(testdataDir, dir, u.RootDir(), "graph."+tcName), graphOutput)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func output(base string, v interface{}) error {
	dst := base
	if *updateExp {
		dst += ".exp.json"
	} else {
		dst += ".got.json"
	}

	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}

	data = append(data, '\n')
	err = ioutil.WriteFile(dst, data, 0700)
	if err != nil {
		return err
	}

	if !*updateExp {
		// Find diff.
		expData, err := ioutil.ReadFile(base + ".exp.json")
		if err != nil {
			return err
		}

		diff, err := diff(expData, data)
		if err != nil {
			return err
		}
		if len(diff) > 0 {
			return fmt.Errorf("%s", diff)
		}
	}

	return nil
}

// unmarshalAsUntyped marshals orig, which is usually a struct, into JSON and
// then to a map[string]interface{}.
func unmarshalAsUntyped(orig interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(orig)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	err = json.Unmarshal(data, &m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func defaultBase(path string) string {
	p, err := build.Default.Import(path, "", build.FindOnly)
	if err != nil {
		return "."
	}
	return p.Dir
}

// isDir returns true if path is an existing directory, and false otherwise.
func isDir(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}

// isFile returns true if path is an existing file, and false otherwise.
func isFile(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.Mode().IsRegular()
}

func diff(b1, b2 []byte) (data []byte, err error) {
	f1, err := ioutil.TempFile("", "sg-toolchain-diff")
	if err != nil {
		return nil, err
	}
	defer os.Remove(f1.Name())
	defer f1.Close()

	f2, err := ioutil.TempFile("", "sg-toolchain-diff")
	if err != nil {
		return nil, err
	}
	defer os.Remove(f2.Name())
	defer f2.Close()

	f1.Write(b1)
	f2.Write(b2)

	data, err = exec.Command("diff", "-u", f1.Name(), f2.Name()).CombinedOutput()
	if len(data) > 0 {
		// diff exits with a non-zero status when the files don't match.
		// Ignore that failure as long as we get output.
		err = nil
	}
	return
}
