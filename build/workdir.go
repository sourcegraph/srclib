package build

import (
	"os"
	"path/filepath"
)

var WorkDir = filepath.Join(os.TempDir(), "sg")
