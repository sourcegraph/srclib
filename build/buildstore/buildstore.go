package buildstore

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/kr/fs"
	"github.com/sourcegraph/rwvfs"
	"sourcegraph.com/sourcegraph/repo"
)

type BuildDataFileInfo struct {
	FullPath string
	CommitID string
	Path     string
	Size     int64
	ModTime  time.Time
}

type WalkableRWVFS struct{ rwvfs.FileSystem }

func (_ WalkableRWVFS) Join(elem ...string) string { return filepath.Join(elem...) }

func ListDataFiles(vfs WalkableRWVFS, repoURI repo.URI, path string) ([]*BuildDataFileInfo, error) {
	var files []*BuildDataFileInfo
	walker := fs.WalkFS(path, vfs)
	for walker.Step() {
		fi := walker.Stat()
		if fi == nil {
			continue
		}
		if fi.IsDir() {
			continue
		}

		pathInRepo, err := filepath.Rel(string(repoURI), walker.Path())
		if err != nil {
			return nil, err
		}

		parts := strings.SplitN(pathInRepo, "/", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("bad build data file path: %q", walker.Path())
		}
		commitID, path := parts[0], parts[1]

		files = append(files, &BuildDataFileInfo{
			FullPath: walker.Path(),
			CommitID: commitID,
			Path:     path,
			Size:     fi.Size(),
			ModTime:  fi.ModTime(),
		})
	}

	return files, nil
}
