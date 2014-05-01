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

	testRootDir, _ := filepath.Abs("testdata")
	testCases := getTestCases(testRootDir, *match)
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
		t.Fatalf("Could not execute diff due to error %s, diff output: %s", err, string(diffOut))
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
		t.Errorf("failed to compute diff: %s", err)
		return false
	} else {
		t.Logf(brush.Green("PASS").String())
		t.Logf("test case %+v", tcase)
		return true
	}
}

func getTestCases(testdir string, match string) []testCase {
	var testCases []testCase
	walker := fs.Walk(testdir)
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
