// +build ignore

package ruby

import (
	"os"
	"os/exec"
	"path/filepath"
)

var RVMDir = filepath.Join(os.Getenv("HOME"), ".rvm")
var RVM = filepath.Join(RVMDir, "bin/rvm")

func rvmCommand(executable string, args ...string) *exec.Cmd {
	rvmargs := []string{RubyVersion, "do", executable}
	rvmargs = append(rvmargs, args...)
	return exec.Command("rvm", rvmargs...)
}
