package src

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aybabtme/color/brush"
	"github.com/sourcegraph/srclib/buildstore"
	"github.com/sourcegraph/srclib/util2"
)

func init() {
	_, err := CLI.AddCommand("test",
		"run test cases",
		`Tests a tool. If no TREEs are specified, all directories in testdata/case relative to the current directory are
used (except those whose name begins with "_").

Expected and actual outputs for a tree are stored in TREE/../../{expected,actual}/TREEBASE, respectively, where TREEBASE is the basename of TREE.

After making the tree, "src test" compares the actual test output against the expected test output. Any differences trigger a test failure, and the differinglines are printed.

If the --gen flag is used, the expected test output is removed and regenerated. You should regenerate the expected output whenever you make changes to the toolchain that alter the desired output. Be sure to check the new expected output for errors manually; it's easy to accidentally commit new expected output that is incorrect.

CONFIGURING TESTS

Use a Srcfile in trees whose tests you want to configure (e.g., by only running a scanner). There is no special configuration for testing beyond what's possible with Srcfile.

EXAMPLE

For example, suppose you run "src test" in a directory with the following files:

  testdata/case/foo/foo.go

Then the expected test output is assumed to exist at (or will be created at, if -gen is used):

  testdata/expected/foo/*

And the actual test output is written to:

  testdata/actual/foo/*
`,
		&testCmd,
	)
	if err != nil {
		log.Fatal(err)
	}
}

type TestCmd struct {
	GenerateExpected bool `long:"gen" description:"(re)generate expected output for all test cases and exit"`

	Args struct {
		Trees []Directory `name:"TREES" description:"trees to treat as test cases"`
	} `positional-args:"yes"`
}

var testCmd TestCmd

func (c *TestCmd) Execute(args []string) error {
	var trees []string
	if len(c.Args.Trees) > 0 {
		for _, tree := range c.Args.Trees {
			trees = append(trees, string(tree))
		}
	} else {
		entries, err := ioutil.ReadDir("testdata/case")
		if err != nil {
			return err
		}
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), "_") {
				continue
			}
			trees = append(trees, filepath.Join("testdata/case", e.Name()))
		}
	}

	if gopt.Verbose {
		log.Printf("Testing trees: %v", trees)
	}

	for _, tree := range trees {
		if gopt.Verbose {
			log.Printf("Testing tree %v...", tree)
		}
		expectedDir := filepath.Join(tree, "../../expected", filepath.Base(tree))
		actualDir := filepath.Join(tree, "../../actual", filepath.Base(tree))
		if err := testTree(tree, expectedDir, actualDir, c.GenerateExpected); err != nil {
			return fmt.Errorf("testing tree %q: %s", tree, err)
		}
	}

	return nil
}

func testTree(treeDir, expectedDir, actualDir string, generate bool) error {
	treeName := filepath.Base(treeDir)
	if treeName == "." {
		absTreeDir, err := filepath.Abs(treeDir)
		if err != nil {
			return err
		}
		treeName = filepath.Base(absTreeDir)
	}

	// Determine and wipe the desired output dir.
	var outputDir string
	if generate {
		outputDir = expectedDir
	} else {
		outputDir = actualDir
	}
	outputDir, _ = filepath.Abs(outputDir)
	if err := os.RemoveAll(outputDir); err != nil {
		return err
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	// Symlink ${treeDir}/.srclib-cache/${commitID} to the desired output dir.
	//
	// TODO(sqs): make `src make` not necessarily write to a .srclib-cache/...
	// path containing the commit ID. When we're just making a tree, we don't
	// know or care about the commit ID.
	treeRepo, err := OpenRepo(treeDir)
	if err != nil {
		return err
	}
	origOutputDestDir := filepath.Join(treeDir, buildstore.BuildDataDirName, treeRepo.CommitID)
	if err := os.Mkdir(filepath.Dir(origOutputDestDir), 0755); err != nil && !os.IsExist(err) {
		return err
	}
	if err := os.Remove(origOutputDestDir); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.Symlink(outputDir, origOutputDestDir); err != nil {
		return err
	}

	// Remove the symlink when we're done so the repo doesn't have
	// uncommitted changes.
	defer os.Remove(origOutputDestDir)

	// Run `src make`.
	var w io.Writer
	var buf bytes.Buffer
	if testing.Verbose() {
		w = io.MultiWriter(&buf, os.Stderr)
	} else {
		w = &buf
	}
	cmd := exec.Command("src", "-v", "make", "-m", "docker")
	cmd.Dir = treeDir
	cmd.Stderr, cmd.Stdout = w, w

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Command %v in %s failed: %s.\n\nOutput was:\n%s", cmd.Args, treeName, err, buf.String())
	}

	if generate {
		return fmt.Errorf("Successfully generated expected output for %s in %s. (Triggering an error so you won't mistakenly interpret a 0 return code as a test success. Run without -gen to actually run the test.)", treeName, expectedDir)
	}
	return checkResults(buf, treeDir, actualDir, expectedDir)
}

func checkResults(output bytes.Buffer, treeDir, actualDir, expectedDir string) error {
	treeName := filepath.Base(treeDir)
	out, err := exec.Command("diff", "-ur", expectedDir, actualDir).CombinedOutput()
	if err != nil || len(out) > 0 {
		fmt.Println(brush.Red(treeName + " FAIL").String())
		fmt.Printf("Diff failed for %s: %s.", treeName, err)
		if len(out) > 0 {
			fmt.Println(brush.Red(treeName + " FAIL"))
			fmt.Println(output.String())
			fmt.Println(string(util2.ColorizeDiff(out)))
		}
		return fmt.Errorf("Output for %s differed from expected.", treeName)
	} else {
		fmt.Println(brush.Green(treeName + " PASS").String())
	}
	return nil
}
