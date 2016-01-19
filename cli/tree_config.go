package cli

// TreeCache is a handle to the on-disk cache of srclib data generated
// for a source tree.
type TreeCache struct {
	// Root directory containing source tree being analyzed
	RootDir string

	// TreeType is the source tree type (e.g., version control
	// repositories like git and hg or package manager repositories
	// like pip packages and npm modules). May be empty if the type
	// cannot be determined straightaway.
	TreeType string

	// Version of the tree cache if the source tree contains multiple
	// versions.
	Version string
}

// OpenTreeCache returns a handle to the tree cache rooted at
// directory dir. The tree type can optionally be passed as an
// argument (the tree type can only be inferred in some cases).
func OpenTreeCache(dir, treeType string) (*TreeCache, error) {
	if repo, err := OpenRepo(dir); err == nil {
		if treeType == "" {
			treeType = repo.VCSType
		}
		return &TreeCache{RootDir: repo.RootDir, TreeType: treeType, Version: repo.CommitID}, nil
	}
	return &TreeCache{RootDir: dir, TreeType: treeType}, nil
}
