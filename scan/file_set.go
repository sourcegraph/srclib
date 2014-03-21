package scan

// FileSet represents a set of files underneath a directory.
type FileSet struct {
	// Dir is the root directory of this FileSet.
	Dir string

	// Files is a map of FileInfo keyed on the relative path of each file in
	// this FileSet.
	Files map[string]*FileInfo
}

// FilesToAnalyze returns a list of files (with paths relative to u.Dir) whose
// Analyze field is true.
func (u *FileSet) FilesToAnalyze() (paths []string) {
	for path, f := range u.Files {
		if f.Analyze {
			paths = append(paths, path)
		}
	}
	return
}

// FilesToBlame returns a list of files (with paths relative to u.Dir) whose
// Blame field is true.
func (u *FileSet) FilesToBlame() (paths []string) {
	for path, f := range u.Files {
		if f.Blame {
			paths = append(paths, path)
		}
	}
	return
}

// FileSets implements sort.Interface.
type FileSets []FileSet

func (u FileSets) Len() int           { return len(u) }
func (u FileSets) Swap(i, j int)      { u[i], u[j] = u[j], u[i] }
func (u FileSets) Less(i, j int) bool { return u[i].Dir < u[j].Dir }
