package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kr/fs"
	"sourcegraph.com/sourcegraph/rwvfs"
	"sourcegraph.com/sourcegraph/srclib/buildstore"
	"sourcegraph.com/sourcegraph/srclib/unit"
)

// ReadCached reads a Tree's configuration from all of its source unit
// definition files in the .srclib-cache directory underneath dir. It does not
// read the Srcfile; the Srcfile's directives are already baked into the cached
// source unit definition files.
func ReadCached(buildStore *buildstore.RepositoryStore, commitID string) (*Tree, error) {
	buildDataDir, err := buildstore.BuildDir(buildStore, commitID)
	if err != nil {
		return nil, err
	}

	// Get all .srclib-cache/**/*.unit.v0.json files.
	var unitFiles []string
	unitSuffix := buildstore.DataTypeSuffix(unit.SourceUnit{})
	dataPath := buildStore.CommitPath(commitID)
	var w *fs.Walker
	needsCommitPrefix := false
	if fi, err := buildStore.Lstat(dataPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("build cache dir does not exist. Did you run `src config`? [dataPath=%q]", dataPath)
	} else if err != nil {
		return nil, err
	} else if fi.Mode().IsDir() {
		w = fs.WalkFS(dataPath, buildStore)
	} else if fi.Mode()&os.ModeSymlink > 0 {
		dst, err := os.Readlink(buildDataDir)
		if err != nil {
			return nil, err
		}
		w = fs.WalkFS(".", walkableFileSystem{rwvfs.OS(dst)})
		needsCommitPrefix = true
	} else {
		return nil, fmt.Errorf("invalid build cache dir")
	}
	for w.Step() {
		if strings.HasSuffix(w.Path(), unitSuffix) {
			path := w.Path()
			if needsCommitPrefix {
				path = filepath.Join(commitID, path)
			}
			unitFiles = append(unitFiles, path)
		}
	}

	// Parse units
	sort.Strings(unitFiles)
	units := make([]*unit.SourceUnit, len(unitFiles))
	for i, unitFile := range unitFiles {
		f, err := buildStore.Open(unitFile)
		if err != nil {
			return nil, err
		}
		var u *unit.SourceUnit
		if err := json.NewDecoder(f).Decode(&u); err != nil {
			f.Close()
			return nil, err
		}
		if err := f.Close(); err != nil {
			return nil, err
		}
		units[i] = u
	}

	return &Tree{SourceUnits: units}, nil
}

type walkableFileSystem struct{ rwvfs.FileSystem }

func (_ walkableFileSystem) Join(elem ...string) string { return filepath.Join(elem...) }
