/*
 * Script that runs srcgraph make -test on a set of test repositories
 */
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aybabtme/color/brush"
)

type TestCase struct {
	Name     string
	CloneURL string
	CommitID string
}

var tests = []TestCase{
	{"go-sample-0", "https://github.com/sgtest/go-sample-0", "1dd4664fec342c0727850380931429a5850a4402"},
	{"python-sample-0", "https://github.com/sgtest/python-sample-0", "7f4959756bdfc406f318fedfe9f8e8ba98dfe48b"},
	{"python-sample-1", "https://github.com/sgtest/python-sample-1", "8a7dac432187679e8a009c682aa9c90640ff3051"},
	{"javascript-nodejs-sample-0", "https://github.com/sgtest/javascript-nodejs-sample-0", "e10faf45fd536676a48bbbdb6ab650e7721782bb"},
	{"javascript-nodejs-xrefs-0", "https://github.com/sgtest/javascript-nodejs-xrefs-0", "a82948d15bfcbac86530caf0e9c0929e6c41c353"},
}

var verbose = flag.Bool("verbose", false, "if verbose is true, print individual test output")
var list = flag.Bool("list", false, "list all test cases")
var match = flag.String("match", "", "match cases (by name substring)")

func main() {
	flag.Parse()
	if *list {
		for _, test := range tests {
			fmt.Printf("%s (%s:%s)\n", test.Name, test.CloneURL, test.CommitID)
		}
		os.Exit(0)
	}

	var exitCode = 0
	testDir, err := ioutil.TempDir("", "sg-test")
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}
	defer os.RemoveAll(testDir)

	for _, test := range tests {
		if *match != "" && !strings.Contains(test.Name, *match) {
			continue
		}

		fmt.Printf("Testing %s...\n", test.Name)
		err := testRepo(test, testDir)
		if err != nil {
			exitCode = 1
			fmt.Printf(brush.Red("Test %s failed\n").String(), test.Name)
		} else {
			fmt.Printf(brush.Green("Test %s succeeded\n").String(), test.Name)
		}
	}

	os.Exit(exitCode)
}

func testRepo(t TestCase, testDir string) error {
	{
		cloneCmd := exec.Command("git", "clone", t.CloneURL, t.Name)
		cloneCmd.Dir = testDir
		if *verbose {
			cloneCmd.Stdout, cloneCmd.Stderr = os.Stdout, os.Stderr
		}
		err := cloneCmd.Run()
		if err != nil {
			return err
		}
	}
	{
		checkoutCmd := exec.Command("git", "checkout", t.CommitID)
		checkoutCmd.Dir = filepath.Join(testDir, t.Name)
		if *verbose {
			checkoutCmd.Stdout, checkoutCmd.Stderr = os.Stdout, os.Stderr
		}
		err := checkoutCmd.Run()
		if err != nil {
			return fmt.Errorf("Error (%s)", err)
		}
	}
	{
		srcgraphCmd := exec.Command("srcgraph", "make", "-test")
		srcgraphCmd.Dir = filepath.Join(testDir, t.Name)
		if *verbose {
			srcgraphCmd.Stdout, srcgraphCmd.Stderr = os.Stdout, os.Stderr
		}
		err := srcgraphCmd.Run()
		if err != nil {
			return fmt.Errorf("Error (%s)", err)
		}
	}
	return nil
}
