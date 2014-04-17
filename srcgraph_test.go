package srcgraph

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
	"sourcegraph.com/sourcegraph/util"

	"github.com/kr/fs"
)

var keepTestFiles = flag.Bool("keep", false, "keep test files around after test is run")
var generate = flag.Bool("generate", false, "generate new expected test data")

func Test_SrcgraphCmd(t *testing.T) {
	testdir, _ := filepath.Abs("testdata")
	testCases := getTestCases(testdir)
	testwd, _ := os.Getwd()

	for _, testCase := range testCases {
		t.Logf("Running test case %+v", testCase)
		err := os.Chdir(testCase.Dir)
		if err != nil {
			t.Fatalf("Could not chdir: %s", err)
		}

		cmd := []string{"-v", "-commit=expected-test"}
		if !*generate {
			cmd = append(cmd, "-test")
			if *keepTestFiles {
				cmd = append(cmd, "-test-keep")
			}
		}
		make_(cmd)
	}
	os.Chdir(testwd)
}

type testCase struct {
	Dir string
}

func getTestCases(testdir string) []testCase {
	var testCases []testCase
	walker := fs.Walk(testdir)
	for walker.Step() {
		path := walker.Path()
		if walker.Stat().IsDir() && util.IsFile(filepath.Join(path, ".git/config")) {
			testCases = append(testCases, testCase{Dir: path})
		}
	}
	return testCases
}
