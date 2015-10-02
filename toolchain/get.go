package toolchain

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"sourcegraph.com/sourcegraph/srclib"
)

// CloneOrUpdate downloads the toolchain named by the toolchain path
// (if it does not already exist in the SRCLIBPATH). If update is
// true, it uses the network to update the toolchain.
//
// Assumes that the clone URL is "https://" + path + ".git".
func CloneOrUpdate(path string, update bool) (*Info, error) {
	path = filepath.Clean(path)

	dir := filepath.SplitList(srclib.Path)[0]
	toolchainDir := filepath.Join(dir, path)

	if fi, err := os.Stat(toolchainDir); os.IsNotExist(err) {
		cloneURL := "https://" + path + ".git"
		cmd := exec.Command("git", "clone", cloneURL, toolchainDir)
		cmd.Stdout, cmd.Stderr = os.Stderr, os.Stderr
		if err := cmd.Run(); err != nil {
			return nil, err
		}
	} else if update && fi.Mode().IsDir() {
		cmd := exec.Command("git", "pull", "origin", "master")
		cmd.Dir = toolchainDir
		cmd.Stdout, cmd.Stderr = os.Stderr, os.Stderr
		if err := cmd.Run(); err != nil {
			return nil, err
		}
	} else {
		return nil, err
	}

	tc, err := Lookup(path)
	if err != nil {
		return nil, fmt.Errorf("clone-or-update toolchain failed: %s (is %s a srclib toolchain repository?)", err, path)
	}
	return tc, nil
}
