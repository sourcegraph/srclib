package toolchain

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"sourcegraph.com/sourcegraph/srclib"
)

type AddOpt struct {
	// Force add a toolchain, overwriting any existing toolchains.
	Force bool
}

// Add creates a symlink in the SRCLIBPATH so that the toolchain in dir is
// available at the toolchainPath.
func Add(dir, toolchainPath string, opt *AddOpt) error {
	if opt == nil {
		opt = &AddOpt{}
	}
	if !opt.Force {
		if _, err := Lookup(toolchainPath); !os.IsNotExist(err) {
			return fmt.Errorf("a toolchain already exists at toolchain path %q", toolchainPath)
		}
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	srclibpathEntry := strings.SplitN(srclib.Path, ":", 2)[0]
	targetDir := filepath.Join(srclibpathEntry, toolchainPath)

	//src toolchain add should check current dir before adding #98 @jelmerdereus
  if _, err := os.Stat(filepath.Join(targetDir, ConfigFilename)); os.IsNotExist(err) {
    return fmt.Errorf("No suitable target directory:\n %s\n", err.Error())
  }

	if err := os.MkdirAll(filepath.Dir(targetDir), 0700); err != nil {
		return err
	}

	if !opt.Force {
		return os.Symlink(absDir, targetDir)
	}
	// Force install the toolchain by removing the directory if
	// the symlink fails, and then try the symlink again.
	if err := os.Symlink(absDir, targetDir); err != nil {
		if err := os.RemoveAll(targetDir); err != nil {
			return err
		}
		return os.Symlink(absDir, targetDir)
	}
	return nil
}
