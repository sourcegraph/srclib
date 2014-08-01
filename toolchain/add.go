package toolchain

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"sourcegraph.com/sourcegraph/srclib"
)

// Add creates a symlink in the SRCLIBPATH so that the toolchain in dir is
// available at the toolchainPath.
func Add(dir, toolchainPath string) error {
	if _, err := Lookup(toolchainPath); !os.IsNotExist(err) {
		return fmt.Errorf("a toolchain already exists at toolchain path %q", toolchainPath)
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	srclibpathEntry := strings.SplitN(srclib.Path, ":", 2)[0]
	targetDir := filepath.Join(srclibpathEntry, toolchainPath)

	if err := os.MkdirAll(filepath.Dir(targetDir), 0700); err != nil {
		return err
	}

	return os.Symlink(absDir, targetDir)
}
