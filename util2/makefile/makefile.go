package makefile

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
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

func Makefile(rules []Rule) ([]byte, error) {
	var mf bytes.Buffer
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

func MakeRules(dir string, rules []Rule) error {
	mf, err := Makefile(rules)
	if err != nil {
		return err
	}
	return Make(dir, mf)
}

func Make(dir string, makefile []byte) error {
	tmpFile, err := ioutil.TempFile("", "sg-makefile")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())

	err = ioutil.WriteFile(tmpFile.Name(), makefile, 0600)
	if err != nil {
		return err
	}

	mk := exec.Command("make", "-f", tmpFile.Name(), "-C", dir, "all")
	mk.Stdout = os.Stderr
	mk.Stderr = os.Stderr
	return mk.Run()
}
