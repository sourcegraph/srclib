package toolchain

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"sourcegraph.com/sourcegraph/srclib"
)

// Get downloads the toolchain named by the toolchain path (if it does not
// already exist in the SRCLIBPATH).
//
// Assumes that the clone URL is "https://" + path + ".git".
func Get(path string) (*Info, error) {
	path = filepath.Clean(path)
	if tc, err := Lookup(path); !os.IsNotExist(err) {
		return tc, err
	}

	dir := strings.SplitN(srclib.Path, ":", 2)[0]
	cmd := exec.Command("git", "clone", "https://"+path+".git", filepath.Join(dir, path))
	cmd.Stdout, cmd.Stderr = os.Stderr, os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	tc, err := Lookup(path)
	if err != nil {
		return nil, fmt.Errorf("get toolchain failed: %s (is %s a srclib toolchain repository?)", err, path)
	}
	return tc, nil
}
