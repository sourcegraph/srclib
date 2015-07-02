package toolchain

import (
	"fmt"
	"os"
	"path/filepath"

	"sourcegraph.com/sourcegraph/srclib"
)

// TempDirName is directory under SRCLIBPATH where to store temp directories for toolchains.
const TempDirName = ".tmp"

// TempDir returns toolchains temp directory. Directory is created it doesn't
// exist.
func TempDir(toolchainPath string) (string, error) {
	tc, err := Lookup(toolchainPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf(
				"get toolchain failed: %s (is %s a srclib toolchain repository?)",
				err,
				toolchainPath,
			)
		}
		return "", err
	}

	tmpDir := filepath.Join(srclib.PathEntries()[0], TempDirName, tc.Path)

	if err := os.MkdirAll(tmpDir, 0700); err != nil {
		return "", err
	}

	return tmpDir, nil
}
