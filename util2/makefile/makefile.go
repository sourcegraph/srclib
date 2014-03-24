package makefile

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

type Target interface {
	Name() string
}

type Recipe interface {
	Command() []string
}

type CommandRecipe []string

func (r CommandRecipe) Command() []string { return r }

type Rule interface {
	Target() Target
	Prereqs() []string
	Recipes() []Recipe
}

type Phonier interface {
	Phony() bool
}

func isPhony(r Rule) bool {
	if p, ok := r.(Phonier); ok {
		return p.Phony()
	}
	return false
}

func Makefile(rules []Rule, vars []string) ([]byte, error) {
	var mf bytes.Buffer

	for _, v := range vars {
		fmt.Fprintln(&mf, v)
	}
	if len(vars) > 0 {
		fmt.Fprintln(&mf)
	}

	var all, phonies []string

	for _, rule := range rules {
		ruleName := rule.Target().Name()
		all = append(all, ruleName)
		if isPhony(rule) {
			phonies = append(phonies, ruleName)
		}
	}
	if len(all) > 0 {
		fmt.Fprintf(&mf, "all: %s\n", strings.Join(all, " "))
	}
	if len(phonies) > 0 {
		fmt.Fprintf(&mf, "\n.PHONY: all %s\n", strings.Join(phonies, " "))
	}

	for _, rule := range rules {
		fmt.Fprintln(&mf)

		ruleName := rule.Target().Name()
		fmt.Fprintf(&mf, "%s:", ruleName)
		for _, prereq := range rule.Prereqs() {
			fmt.Fprintf(&mf, " %s", prereq)
		}
		fmt.Fprintln(&mf)
		for _, recipe := range rule.Recipes() {
			fmt.Fprintf(&mf, "\t%s\n", strings.Join(recipe.Command(), " "))
		}
	}

	return mf.Bytes(), nil
}

func MakeRules(dir string, rules []Rule, vars []string, args []string) error {
	mf, err := Makefile(rules, vars)
	if err != nil {
		return err
	}
	return Make(dir, mf, args)
}

func Make(dir string, makefile []byte, args []string) error {
	tmpFile, err := ioutil.TempFile("", "sg-makefile")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())

	err = ioutil.WriteFile(tmpFile.Name(), makefile, 0600)
	if err != nil {
		return err
	}

	args = append(args, "-f", tmpFile.Name(), "-C", dir)
	mk := exec.Command("make", args...)
	mk.Stdout = os.Stderr
	mk.Stderr = os.Stderr
	return mk.Run()
}

var cleanRE = regexp.MustCompile(`^[\w\d_/.-]+$`)

func Quote(s string) string {
	if cleanRE.MatchString(s) {
		return s
	}
	q := strconv.Quote(s)
	return "'" + strings.Replace(q[1:len(q)-1], "'", "", -1) + "'"
}
