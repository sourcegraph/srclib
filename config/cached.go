package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"code.google.com/p/rog-go/parallel"

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
		// Symlinks are currently used by the `src test` command for
		// the .srclib-cache dirs of test case repos. We assume that
		// symlinks only exist in OS VFSes, and if we see a symlink,
		// we dereference it and open its target using an OS VFS. This
		// will break if other VFSes have symlinks.
		buildDataDir, err := buildstore.BuildDir(buildStore, commitID)
		if err != nil {
			return nil, err
		}

		dst, err := os.Readlink(buildDataDir)
		if err != nil {
			return nil, err
		}
		w = fs.WalkFS(".", rwvfs.Walkable(rwvfs.OS(dst)))
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
	par := parallel.NewRun(runtime.GOMAXPROCS(0))
	for i_, unitFile_ := range unitFiles {
		i, unitFile := i_, unitFile_
		par.Do(func() error {
			f, err := buildStore.Open(unitFile)
			if err != nil {
				return err
			}
			if err := json.NewDecoder(f).Decode(&units[i]); err != nil {
				f.Close()
				return err
			}
			if err := f.Close(); err != nil {
				return err
			}
			return nil
		})
	}
	if err := par.Wait(); err != nil {
		return nil, err
	}

	return &Tree{SourceUnits: units}, nil
}
