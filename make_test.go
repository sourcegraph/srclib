package srcgraph

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"sourcegraph.com/sourcegraph/srcgraph/buildstore"
	"sourcegraph.com/sourcegraph/srcgraph/util2"

	"testing"

	"github.com/aybabtme/color/brush"
)

var mode = flag.String("test.mode", "test", "[test|keep|gen] 'test' runs test as normal; keep keeps around generated test files for inspection after tests complete; 'gen' generates new expected test data")
var repoMatch = flag.String("test.repo", "", "only test `srcgraph make` for repos that contain this string")

func TestMakeCmd(t *testing.T) {
	if testing.Short() {
		t.Skip("srcgraph make tests take a long time; skipping for -test.short")
	}

	if *repoMatch != "" {
		t.Logf("Testing `srcgraph make` on repositories that contain \"%s\"", *repoMatch)
	}

	// Since we exec `srcgraph`, make sure it's up-to-date.
	if out, err := exec.Command("make", "-C", "..", "srcgraph").CombinedOutput(); err != nil {
		t.Errorf("Failed to build srcgraph for `srcgraph make` tests: %s.\n\nOutput was:\n%s", err, out)
		return
	}

	cmd := exec.Command("git", "submodule", "update", "--init", "srcgraph/testdata/repos")
	cmd.Dir = ".."
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Errorf("Failed to update or init git submodules: %s.\n\nOutput was:\n%s", err, out)
		return
	}

	const testReposDir = "testdata/repos"
	fis, err := ioutil.ReadDir("testdata/repos")
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	var wg sync.WaitGroup
	for _, fi := range fis {
		if repoDir := fi.Name(); strings.Contains(repoDir, *repoMatch) {
			fullRepoDir := filepath.Join(testReposDir, repoDir)
			wg.Add(1)
			go func() {
				defer wg.Done()
				testMakeCmd(t, fullRepoDir)
			}()
			n++
		}
	}
	wg.Wait()
	if n == 0 {
		t.Errorf("No TestMakeCmd cases were run. Did none match your constraint -test.match=%q?", *repoMatch)
	}
}

func testMakeCmd(t *testing.T, repoDir string) {
	repoName := filepath.Base(repoDir)

	// Directories for the actual ("got") and expected ("want") outputs.
	gotOutputDir := filepath.Join(repoDir, "../../repos-output/got", repoName)
	wantOutputDir := filepath.Join(repoDir, "../../repos-output/want", repoName)

	// Determine and wipe the desired output dir.
	var outputDir string
	if *mode == "gen" {
		outputDir = wantOutputDir
	} else {
		outputDir = gotOutputDir
	}
	outputDir, _ = filepath.Abs(outputDir)
	if err := os.RemoveAll(outputDir); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Symlink ${repoDir}/.sourcegraph-data/${commitID} to the desired output dir.
	origOutputDestDir := filepath.Join(repoDir, buildstore.BuildDataDirName, getHEADOrTipCommitID(t, repoDir))
	if err := os.Mkdir(filepath.Dir(origOutputDestDir), 0755); err != nil && !os.IsExist(err) {
		t.Fatal(err)
	}
	if err := os.Remove(origOutputDestDir); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if err := os.Symlink(outputDir, origOutputDestDir); err != nil {
		t.Error(err)
		return
	}
	defer func() {
		// Remove the symlink when we're done so the repo doesn't have
		// uncommitted changes.
		if err := os.Remove(origOutputDestDir); err != nil {
			t.Fatal(err)
		}
	}()

	// Run `srcgraph make`.
	var w io.Writer
	var buf bytes.Buffer
	if testing.Verbose() {
		w = io.MultiWriter(&buf, os.Stderr)
	} else {
		w = &buf
	}
	cmd := exec.Command("srcgraph", "-v", "make")
	cmd.Stderr, cmd.Stdout = w, w
	cmd.Dir = repoDir

	err := cmd.Run()
	if err != nil {
		t.Errorf("Command %v in %s failed: %s.\n\nOutput was:\n%s", cmd.Args, repoDir, err, buf.String())
		return
	}

	if *mode == "gen" {
		t.Errorf("Successfully generated expected output for %s in %s. (Triggering test error so you won't mistakenly interpret a 0 return code as a test success. Run without -test.mode=gen to run the test.)", repoName, wantOutputDir)
	} else {
		checkResults(t, buf, repoName, gotOutputDir, wantOutputDir)
	}
}

func checkResults(t *testing.T, output bytes.Buffer, repoName, gotOutputDir, wantOutputDir string) {
	out, err := exec.Command("diff", "-ur", wantOutputDir, gotOutputDir).CombinedOutput()
	if err != nil || len(out) > 0 {
		t.Logf(brush.Red(repoName + " FAIL").String())
		t.Errorf("Diff failed for %s: %s.", repoName, err)
		if len(out) > 0 {
			fmt.Println(brush.Red(repoName + "FAIL"))
			fmt.Println(output.String())
			fmt.Println(string(util2.ColorizeDiff(out)))
			t.Errorf("Output for %s differed from expected.", repoName)
		}
	} else {
		t.Logf(brush.Green(repoName + " PASS").String())
	}
}

func getHEADOrTipCommitID(t *testing.T, repoDir string) string {
	// TODO(sqs): assumes git
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	return string(bytes.TrimSpace(out))
}
