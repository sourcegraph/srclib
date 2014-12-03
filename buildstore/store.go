package buildstore

import (
	"os"
	"path/filepath"

	"github.com/kr/fs"

	"sort"

	"sourcegraph.com/sourcegraph/rwvfs"
)

// BuildDataDirName is the name of the directory in which local
// repository build data is stored, relative to the top-level dir of a
// VCS repository.
var BuildDataDirName = ".srclib-cache"

// A MultiStore contains RepoBuildStores for multiple repositories.
type MultiStore struct {
	fs rwvfs.WalkableFileSystem
}

// NewMulti creates a new multi-repo build store.
func NewMulti(fs rwvfs.FileSystem) *MultiStore {
	return &MultiStore{rwvfs.Walkable(fs)}
}

func (s *MultiStore) RepoBuildStore(repoURI string) (RepoBuildStore, error) {
	path := filepath.Clean(string(repoURI))
	return Repo(rwvfs.Walkable(rwvfs.Sub(s.fs, path))), nil
}

// A RepoBuildStore stores and exposes a repository's build data in a
// VFS.
type RepoBuildStore interface {
	// Commit returns a VFS for accessing and writing build data for a
	// specific commit.
	Commit(commitID string) rwvfs.WalkableFileSystem

	// FilePath returns the path (from the repo build store's root) to
	// a file at the specified commit ID.
	FilePath(commitID string, file string) string
}

// Repo creates a new single-repository build store rooted at the
// given filesystem.
func Repo(repoStoreFS rwvfs.WalkableFileSystem) RepoBuildStore {
	return &repoBuildStore{repoStoreFS}
}

// LocalRepo creates a new single-repository build store for the VCS
// repository whose top-level directory is repoDir.
//
// The store is laid out as follows:
//
//   .                the root dir of repoStoreFS
//   <COMMITID>/**/*  build data for a specific commit
func LocalRepo(repoDir string) (RepoBuildStore, error) {
	storeDir := filepath.Join(repoDir, BuildDataDirName)
	if err := os.Mkdir(storeDir, 0700); err != nil && !os.IsExist(err) {
		return nil, err
	}
	return Repo(rwvfs.Walkable(rwvfs.OS(storeDir))), nil
}

type repoBuildStore struct {
	fs rwvfs.WalkableFileSystem
}

func (s *repoBuildStore) Commit(commitID string) rwvfs.WalkableFileSystem {
	return rwvfs.Walkable(rwvfs.Sub(s.fs, s.commitPath(commitID)))
}

func (s *repoBuildStore) commitPath(commitID string) string { return commitID }

func (s *repoBuildStore) FilePath(commitID, path string) string {
	return filepath.Join(s.commitPath(commitID), path)
}

// RemoveAllDataForCommit removes all files and directories from the
// repo build store for the given commit.
func RemoveAllDataForCommit(s RepoBuildStore, commitID string) error {
	commitFS := s.Commit(commitID)
	w := fs.WalkFS(".", commitFS)
	var dirs []string // remove dirs after removing all files
	for w.Step() {
		if err := w.Err(); err != nil {
			return err
		}
		if w.Stat().IsDir() {
			dirs = append(dirs, w.Path())
		} else {
			if err := commitFS.Remove(w.Path()); err != nil {
				return err
			}
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(dirs))) // reverse so we delete leaf dirs first
	for _, dir := range dirs {
		if err := commitFS.Remove(dir); err != nil {
			return err
		}
	}
	return nil
}

func BuildDataExistsForCommit(s RepoBuildStore, commitID string) (bool, error) {
	cfs := s.Commit(commitID)
	_, err := cfs.Stat(".")
	if err == nil {
		return true, nil
	}
	return false, err
}
