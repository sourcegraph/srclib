package src

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aybabtme/color/brush"
	"github.com/kr/fs"
	"sourcegraph.com/sourcegraph/srclib/authorship"
	"sourcegraph.com/sourcegraph/srclib/buildstore"
	"sourcegraph.com/sourcegraph/srclib/util"
	"sourcegraph.com/sourcegraph/srclib/vcsutil"
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

	CheckInternalTargets bool `long:"check-internal-targets" description:"also produce and check internal command outputs (ex: blame, authorship)"`

	ToolchainExecOpt

	Args struct {
		Trees []Directory `name:"TREES" description:"trees to treat as test cases"`
	} `positional-args:"yes"`
}

var testCmd TestCmd

func (c *TestCmd) Execute(args []string) error {
	exeMethods := strings.Split(c.ExeMethods, ",")
	if len(exeMethods) == 0 {
		return errors.New("At least one toolchain execution method must be specified (with -m or --methods).")
	}

	for _, exeMethod := range exeMethods {
		if GlobalOpt.Verbose {
			log.Printf("Executing tests using method: %s", exeMethod)
		}

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

		if GlobalOpt.Verbose {
			log.Printf("Testing trees: %v", trees)
		}

		for _, tree := range trees {
			if GlobalOpt.Verbose {
				log.Printf("Testing tree %v...", tree)
			}
			expectedDir := filepath.Join(tree, "../../expected", exeMethod, filepath.Base(tree))
			actualDir := filepath.Join(tree, "../../actual", exeMethod, filepath.Base(tree))
			if err := testTree(tree, expectedDir, actualDir, exeMethod, c.GenerateExpected, c.CheckInternalTargets); err != nil {
				return fmt.Errorf("testing tree %q: %s", tree, err)
			}
		}
	}

	if c.GenerateExpected {
		log.Fatal("\nSuccessfully wrote expected test output files. Exiting with nonzero return code so you won't mistakenly interpret a 0 return code as a test success. Run without --gen to actually run the test.")
	}

	return nil
}

func testTree(treeDir, expectedDir, actualDir string, exeMethod string, generateExpected, checkInternalTargets bool) error {
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
	if generateExpected {
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
	if GlobalOpt.Verbose {
		w = io.MultiWriter(&buf, os.Stderr)
	} else {
		w = &buf
	}
	cmd := exec.Command("src", "-v", "do-all", "-m", exeMethod)
	cmd.Dir = treeDir
	cmd.Stderr, cmd.Stdout = w, w

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Command %v in %s failed: %s.\n\nOutput was:\n%s", cmd.Args, treeName, err, buf.String())
	}

	if !checkInternalTargets {
		// remove all internal target output files
		internalOutputs := []interface{}{&authorship.SourceUnitOutput{}, &vcsutil.BlameOutput{}}
		w := fs.Walk(outputDir)
		for w.Step() {
			if w.Err() != nil {
				return w.Err()
			}
			for _, o := range internalOutputs {
				if strings.HasSuffix(w.Path(), buildstore.DataTypeSuffix(o)) {
					if err := os.Remove(w.Path()); err != nil {
						return err
					}
					break
				}
			}
		}
	}

	if generateExpected {
		log.Printf("Successfully generated expected output for %s in %s.", treeName, expectedDir)
		return nil
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
			fmt.Println(string(util.ColorizeDiff(out)))
		}
		return fmt.Errorf("Output for %s differed from expected.", treeName)
	} else {
		fmt.Println(brush.Green(treeName + " PASS").String())
	}
	return nil
}
