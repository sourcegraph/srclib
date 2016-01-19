package config

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"

	"github.com/kr/fs"
	"github.com/rogpeppe/rog-go/parallel"
	"golang.org/x/tools/godoc/vfs"
	"sourcegraph.com/sourcegraph/rwvfs"
	"sourcegraph.com/sourcegraph/srclib/buildstore"
	"sourcegraph.com/sourcegraph/srclib/graph2"
)

// ReadCached2 reads a Tree's configuration from all of its build unit
// definition files (which may either be in a local VFS rooted at a
// .srclib-cache/<COMMITID> dir, or a remote VFS). It does not read
// the Srcfile; the Srcfile's directives are already accounted for in
// the cached source unit definition files.
//
// bdfs should be a VFS obtained from a call to
// (buildstore.RepoBuildStore).Commit.
func ReadCached2(bdfs vfs.FileSystem) (*Tree2, error) {
	if _, err := bdfs.Lstat("."); os.IsNotExist(err) {
		return nil, fmt.Errorf("build cache dir does not exist (did you run `srclib config` to create it)?")
	} else if err != nil {
		return nil, err
	}

	// Collect all **/*.unit.json files.
	var unitFiles []string
	unitSuffix := buildstore.DataTypeSuffix(graph2.Unit{})
	w := fs.WalkFS(".", rwvfs.Walkable(rwvfs.ReadOnly(bdfs)))
	for w.Step() {
		if err := w.Err(); err != nil {
			return nil, err
		}
		if path := w.Path(); strings.HasSuffix(path, unitSuffix) {
			unitFiles = append(unitFiles, path)
		}
	}

	// Parse units
	sort.Strings(unitFiles)
	units := make([]*graph2.Unit, len(unitFiles))
	par := parallel.NewRun(runtime.GOMAXPROCS(0))
	for i_, unitFile_ := range unitFiles {
		i, unitFile := i_, unitFile_
		par.Do(func() error {
			f, err := bdfs.Open(unitFile)
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
	return &Tree2{Units: units}, nil
}
