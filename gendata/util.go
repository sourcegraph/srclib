package gendata

import (
	"fmt"
	"path/filepath"
)

// hierarchicalNames returns a slice of hierarchical filenames with branching factor at each
// level specified by structure
func hierarchicalNames(nodeRoot string, leafRoot string, prefix string, structure []int) (filenames []string) {
	if len(structure) == 0 {
		return nil
	}
	if len(structure) == 1 {
		nfiles := structure[0]
		for i := 0; i < nfiles; i++ {
			filenames = append(filenames, filepath.Join(prefix, fmt.Sprintf("%s_%d", leafRoot, i)))
		}
		return filenames
	}

	head, tail := structure[0], structure[1:]
	for i := 0; i < head; i++ {
		subdir := filepath.Join(prefix, fmt.Sprintf("%s_%d", nodeRoot, i))
		filenames = append(filenames, hierarchicalNames(nodeRoot, leafRoot, subdir, tail)...)
	}
	return filenames
}
