package srcgraph

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"sourcegraph.com/sourcegraph/srcgraph/buildstore"
	"sourcegraph.com/sourcegraph/srcgraph/task2"
	"sourcegraph.com/sourcegraph/util"

	"github.com/aybabtme/color/brush"
	"github.com/kr/fs"
	"github.com/sourcegraph/makex"
)

var mode = flag.String("mode", "test", "[test|keep|gen] 'test' runs test as normal; keep keeps around generated test files for inspection after tests complete; 'gen' generates new expected test data")
var match = flag.String("match", "", "run only test cases that contain this string")

func Test_SrcgraphCmd(t *testing.T) {
	actDir := buildstore.BuildDataDirName
	expDir := ".sourcegraph-data-exp"
	if *mode == "gen" {
		buildstore.BuildDataDirName = expDir
	}

	testCases := getTestCases(t, *match)
	allPass := true
	for _, tcase := range testCases {
		func() {
			prevwd, _ := os.Getwd()
			os.Chdir(tcase.Dir)
			defer os.Chdir(prevwd)

			if *mode == "test" {
				defer os.RemoveAll(buildstore.BuildDataDirName)
			}

			t.Logf("Running test case %+v", tcase)
			context, err := NewJobContext(".", task2.DefaultContext)
			if err != nil {
				allPass = false
				t.Errorf("Failed to get job context due to error %s", err)
				return
			}
			context.CommitID = "test-commit"
			err = make__(nil, context, &makex.Default, false, *Verbose)
			if err != nil {
				allPass = false
				t.Errorf("Test case %+v returned error %s", tcase, err)
				return
			}
			if *mode != "gen" {
				same := compareResults(t, tcase, expDir, actDir)
				if !same {
					allPass = false
				}
			}
		}()
	}

	if allPass && *mode != "gen" {
		t.Log(brush.Green("ALL CASES PASS").String())
	}
	if *mode == "gen" {
		t.Log(brush.DarkYellow(fmt.Sprintf("Expected test data dumped to %s directories", expDir)))
	}
	if *mode == "keep" {
		t.Log(brush.Cyan(fmt.Sprintf("Test files persisted in %s directories", actDir)))
	}
	t.Logf("Ran test cases %+v", testCases)
}

type testCase struct {
	Dir string
}

func compareResults(t *testing.T, tcase testCase, expDir, actDir string) bool {
	diffOut, err := exec.Command("diff", "-ur", expDir, actDir).CombinedOutput()
	if err != nil {
		t.Fatalf("Diff failed (%s), diff output: %s", err, string(diffOut))
		return false
	}
	if len(diffOut) > 0 {
		diffStr := string(diffOut)
		t.Errorf(brush.Red("FAIL").String())
		t.Errorf("test case %+v", tcase)
		t.Errorf(diffStr)
		t.Errorf("output differed")
		return false
	} else if err != nil {
		t.Errorf(brush.Red("ERROR").String())
		t.Errorf("test case %+v", tcase)
		t.Errorf("diff failed: %s", err)
		return false
	} else {
		t.Logf(brush.Green("PASS").String())
		t.Logf("test case %+v", tcase)
		return true
	}
}

var testInfo = map[string]struct {
	CloneURL string
	CommitID string
}{
	"go-sample-0":     {"https://github.com/sgtest/go-sample-0", "7538a5ec55397101dae8e099a6c9af53fe06dfdd"},
	"python-sample-0": {"https://github.com/sgtest/python-sample-0", "f873e579e2e4d9d3fb9a30d0694e4a23420b0079"},
}

func getTestCases(t *testing.T, match string) []testCase {
	testRootDir, _ := filepath.Abs("testdata")
	// Pull test repos if necessary
	for testDir, testInfo := range testInfo {
		if !isDir(filepath.Join(testRootDir, testDir, ".git")) {
			t.Logf("Cloning test repository %v into directory %s", testInfo, testDir)
			cloneCmd := exec.Command("git", "clone", testInfo.CloneURL, testDir)
			cloneCmd.Dir = testRootDir
			_, err := cloneCmd.Output()
			if err != nil {
				panic(err)
			}
		}
		ckoutCmd := exec.Command("git", "checkout", testInfo.CommitID)
		ckoutCmd.Dir = filepath.Join(testRootDir, testDir)
		_, err := ckoutCmd.Output()
		if err != nil {
			panic(err)
		}
	}

	// Return test cases
	var testCases []testCase
	walker := fs.Walk(testRootDir)
	for walker.Step() {
		path := walker.Path()
		if walker.Stat().IsDir() && util.IsFile(filepath.Join(path, ".git/config")) {
			if strings.Contains(path, match) {
				testCases = append(testCases, testCase{Dir: path})
			}
		}
	}
	return testCases
}
