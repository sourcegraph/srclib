package buildstore

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"strings"

	"sourcegraph.com/sourcegraph/srcgraph/repo"

	"github.com/kr/fs"
	"github.com/sourcegraph/rwvfs"
)

var BuildDataDirName = ".sourcegraph-data"

var (
	// localDirs stores the OS filesystem path that each local repository store
	// is rooted at. It is used to construct the full, non-VFS path to files
	// within local VFSes.
	localDirs = make(map[*RepositoryStore]string)
)

type MultiStore struct {
	walkableRWVFS
}

func New(fs rwvfs.FileSystem) *MultiStore {
	return &MultiStore{walkableRWVFS{fs}}
}

func (s *MultiStore) RepositoryStore(repoURI repo.URI) (*RepositoryStore, error) {
	path := filepath.Clean(string(repoURI))
	err := rwvfs.MkdirAll(s, path)
	if err != nil {
		return nil, err
	}
	return newRepositoryStore(walkableRWVFS{rwvfs.Sub(s.walkableRWVFS, path)}), nil
}

type RepositoryStore struct {
	walkableRWVFS
}

func newRepositoryStore(fs rwvfs.FileSystem) *RepositoryStore {
	return &RepositoryStore{walkableRWVFS{fs}}
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

	s := newRepositoryStore(walkableRWVFS{rwvfs.OS(storeDir)})

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

type BuildDataFileInfo struct {
	CommitID string
	Path     string
	Size     int64
	ModTime  time.Time
	DataType string
}

func (s *RepositoryStore) CommitPath(commitID string) string { return commitID }

func (s *RepositoryStore) FilePath(commitID, path string) string {
	return filepath.Join(s.CommitPath(commitID), path)
}

func (s *RepositoryStore) ListCommits() ([]string, error) {
	files, err := s.ReadDir(".")
	if err != nil {
		return nil, err
	}

	var commits []string
	for _, f := range files {
		if f.IsDir() {
			commits = append(commits, f.Name())
		}
	}
	return commits, nil
}

const CachedRepositoryConfigFilename = "config.json"

var dataFilesIgnoreBasenames = map[string]struct{}{
	CachedRepositoryConfigFilename: struct{}{},
}

func (s *RepositoryStore) DataFiles(path string) ([]*BuildDataFileInfo, error) {
	var files []*BuildDataFileInfo
	walker := fs.WalkFS(path, s.walkableRWVFS)
	for walker.Step() {
		fi := walker.Stat()
		if fi == nil {
			continue
		}
		if fi.IsDir() {
			continue
		}

		path := strings.TrimPrefix(walker.Path(), "/")

		if _, ignore := dataFilesIgnoreBasenames[filepath.Base(path)]; ignore {
			continue
		}

		parts := strings.SplitN(path, "/", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("bad build data file path: %q", walker.Path())
		}
		commitID, path := parts[0], parts[1]

		dataTypeName, _ := DataType(path)

		files = append(files, &BuildDataFileInfo{
			CommitID: commitID,
			Path:     path,
			Size:     fi.Size(),
			ModTime:  fi.ModTime(),
			DataType: dataTypeName,
		})
	}
	return files, nil
}

func (s *RepositoryStore) DataFilesForCommit(commitID string) ([]*BuildDataFileInfo, error) {
	return s.DataFiles(s.CommitPath(commitID))
}

func (s *RepositoryStore) AllDataFiles() ([]*BuildDataFileInfo, error) {
	return s.DataFiles(".")
}

type walkableRWVFS struct{ rwvfs.FileSystem }

func (_ walkableRWVFS) Join(elem ...string) string { return filepath.Join(elem...) }
