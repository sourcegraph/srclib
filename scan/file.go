package scan

import (
	"os"
	"path/filepath"
)

// FileInfo describes the file and its behavior.
type FileInfo struct {
	// Stat is the result of calling os.Stat on this file.
	Stat os.FileInfo `json:",omitempty"`

	// Languages is the programming/markup/data language that this file
	// contains.
	Language *Language `json:",omitempty"`

	Lib       bool `json:",omitempty"` // library file
	Test      bool `json:",omitempty"` // test file
	Script    bool `json:",omitempty"` // script file
	Example   bool `json:",omitempty"` // example file
	Vendor    bool `json:",omitempty"` // vendor file
	Dist      bool `json:",omitempty"` // dist file (e.g., minified JavaScript or built JARs)
	Generated bool `json:",omitempty"` // generated file

	// Analyze is whether this file should be analyzed by the grapher.
	Analyze bool `json:",omitempty"`

	// Blame is whether this file should be blamed, and the results of blaming
	// be used.
	Blame bool `json:",omitempty"`
}

var SkipTopLevelDirs = []string{".git", ".hg", ".sourcegraph-test"}

func isSkippedTopLevelDir(root, path string, info os.FileInfo) bool {
	if info.IsDir() {
		parent, name := filepath.Split(path)
		parent = filepath.Clean(parent)
		if parent == root {
			for _, skipDir := range SkipTopLevelDirs {
				if name == skipDir {
					return true
				}
			}
		}
	}
	return false
}

// Files scans dir (recursively) for files and returns a FileInfo map keyed
// on each file's relative path.
func Files(dir string) (files map[string]*FileInfo, err error) {
	mustParseLanguages()
	files = make(map[string]*FileInfo)
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if isSkippedTopLevelDir(dir, path, info) {
			return filepath.SkipDir
		}
		if info.Mode().IsRegular() {
			files[path] = &FileInfo{
				Stat:    info,
				Lib:     true,
				Analyze: true,
				Blame:   true,
			}
			if langs := LanguagesByExtension[filepath.Ext(path)]; len(langs) > 0 {
				files[path].Language = langs[0]
			}
		}
		return nil
	})
	return
}
