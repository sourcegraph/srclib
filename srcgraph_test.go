package srcgraph

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"sourcegraph.com/sourcegraph/util"

	"github.com/aybabtme/color/brush"
	"github.com/kr/fs"
)

var keepTestFiles = flag.Bool("keep", false, "keep test files around after test is run")
var generate = flag.Bool("generate", false, "generate new expected test data")
var match = flag.String("match", "", "run only test cases that contain this string")

func Test_SrcgraphCmd(t *testing.T) {
	testdir, _ := filepath.Abs("testdata")
	testCases := getTestCases(testdir, *match)
	testwd, _ := os.Getwd()
	var testCaseErrs = make(map[testCase]error)

	for _, testCase := range testCases {
		t.Logf("Running test case %+v", testCase)
		err := os.Chdir(testCase.Dir)
		if err != nil {
			t.Fatalf("Could not chdir: %s", err)
		}

		cmd := []string{"-v", "-commit=expected-test", "-conf.cache=false"}
		if !*generate {
			cmd = append(cmd, "-test")
			if *keepTestFiles {
				cmd = append(cmd, "-test-keep")
			}
		}

		if err = make_(cmd); err != nil {
			testCaseErrs[testCase] = err
		}
	}
	os.Chdir(testwd)

	t.Logf("Ran test cases %+v", testCases)
	if len(testCaseErrs) == 0 {
		t.Log(brush.Green("** ALL PASS **").String())
	} else {
		t.Log(brush.Red("** ERRORS **").String())
		for testCase, err := range testCaseErrs {
			t.Log(brush.Red(fmt.Sprintf("Test case %+v: %s", testCase, err)).String())
		}
	}
}

type testCase struct {
	Dir string
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
