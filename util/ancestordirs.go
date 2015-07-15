package util

import (
	"path/filepath"
)

// AncestorDirs returns a list of p's ancestor
// directories (optionally including itself) excluding the root ("." or "/")).
func AncestorDirs(p string, self bool) []string {
	if p == "" {
		return nil
	}
	absPath, err := filepath.Abs(p)
	if (err != nil) {
		return nil
	}
	var dirs []string
	if self {
		dirs = append(dirs, absPath)
	}
	dir := filepath.Dir(absPath)
	for dir != "." && dir[len(dir)-1:] != string(filepath.Separator) {
		dirs = append(dirs, dir)
		dir = filepath.Dir(dir)
	}
	return dirs
}

