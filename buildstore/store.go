package buildstore

import (
	"fmt"
	"os"
	"path/filepath"

	"sourcegraph.com/sourcegraph/rwvfs"
)

var BuildDataDirName = ".srclib-cache"

var (
	// localDirs stores the OS filesystem path that each local repository store
	// is rooted at. It is used to construct the full, non-VFS path to files
	// within local VFSes.
	localDirs = make(map[*RepositoryStore]string)
)

type MultiStore struct {
	fs rwvfs.WalkableFileSystem
}

func New(fs rwvfs.FileSystem) *MultiStore {
	return &MultiStore{rwvfs.Walkable(fs)}
}

func (s *MultiStore) RepositoryStore(repoURI string) (*RepositoryStore, error) {
	path := filepath.Clean(string(repoURI))
	return &RepositoryStore{rwvfs.Walkable(rwvfs.Sub(s.fs, path))}, nil
}

type RepositoryStore struct {
	rwvfs.WalkableFileSystem
}

func NewRepositoryStore(repoDir string) (*RepositoryStore, error) {
	storeDir, err := filepath.Abs(filepath.Join(repoDir, BuildDataDirName))

	err = os.Mkdir(storeDir, 0700)
	if os.IsExist(err) {
		err = nil
	}
	if err != nil {
		return nil, err
	}

	s := &RepositoryStore{rwvfs.Walkable(rwvfs.OS(storeDir))}

	localDirs[s] = storeDir

	return s, nil
}

// RootDir returns the OS filesystem path that s's VFS is rooted at, if
// it is a local store (that uses the OS filesystem). If s is a
// non-OS-filesystem VFS, an error is returned.
func RootDir(s *RepositoryStore) (string, error) {
	if dir, present := localDirs[s]; present {
		return dir, nil
	}
	return "", fmt.Errorf("store VFS is not an OS filesystem VFS")
}

func BuildDir(s *RepositoryStore, commitID string) (string, error) {
	rootDataDir, err := RootDir(s)
	if err != nil {
		return "", err
	}
	return filepath.Join(rootDataDir, s.CommitPath(commitID)), nil
}

func FlushCache(s *RepositoryStore, commitID string) error {
	path, err := BuildDir(s, commitID)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(path); err != nil {
		return err
	}
	return nil
}

func (s *RepositoryStore) CommitPath(commitID string) string { return commitID }

func (s *RepositoryStore) FilePath(commitID, path string) string {
	return filepath.Join(s.CommitPath(commitID), path)
}
