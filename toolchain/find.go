package toolchain

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"

	"github.com/kr/fs"
	"sourcegraph.com/sourcegraph/srclib"
)

// Lookup finds a toolchain by path in the SRCLIBPATH. For each DIR in
// SRCLIBPATH, it checks for the existence of DIR/PATH/Srclibtoolchain.
func Lookup(path string) (*Info, error) {
	if noToolchains {
		return nil, nil
	}

	path = filepath.Clean(path)

	matches, err := lookInPaths(filepath.Join(path, ConfigFilename), srclib.Path)
	if err != nil {
		return nil, err
	}

	if len(matches) == 0 {
		return nil, &os.PathError{Op: "toolchain.Lookup", Path: path, Err: os.ErrNotExist}
	}
	if len(matches) > 1 {
		return nil, fmt.Errorf("shadowed toolchain path %q (toolchains: %v)", path, matches)
	}
	return newInfo(path, filepath.Dir(matches[0]), ConfigFilename)
}

// List finds all toolchains in the SRCLIBPATH.
//
// List does not find nested toolchains; i.e., if DIR is a toolchain
// dir (with a DIR/Srclibtoolchain file), then none of DIR's
// subdirectories are searched for toolchains.
func List() ([]*Info, error) {
	if noToolchains {
		return nil, nil
	}

	var found []*Info
	seen := map[string]string{}

	dirs := filepath.SplitList(srclib.Path)

	// maps symlinked trees to their original path
	origDirs := map[string]string{}

	for i := 0; i < len(dirs); i++ {
		dir := dirs[i]
		if dir == "" {
			dir = "."
		}
		w := fs.Walk(dir)
		for w.Step() {
			if w.Err() != nil {
				return nil, w.Err()
			}
			fi := w.Stat()
			name := fi.Name()
			path := w.Path()
			if path != dir && (name[0] == '.' || name[0] == '_') {
				w.SkipDir()
			} else if fi.Mode()&os.ModeSymlink != 0 {
				// Check if symlink points to a directory.
				if sfi, err := os.Stat(path); err == nil {
					if !sfi.IsDir() {
						continue
					}
				} else if os.IsNotExist(err) {
					continue
				} else {
					return nil, err
				}

				targetPath, err := os.Readlink(path)
				if err != nil {
					return nil, err
				}

				if _, traversed := origDirs[targetPath]; !traversed {
					// traverse symlinks but refer to symlinked trees' toolchains using
					// the path to them through the original entry in SRCLIBPATH
					dirs = append(dirs, targetPath)
					origDirs[targetPath], _ = filepath.Rel(dir, path)
				}
			} else if fi.Mode().IsDir() {
				// Check for Srclibtoolchain file in this dir.

				if _, err := os.Stat(filepath.Join(path, ConfigFilename)); os.IsNotExist(err) {
					continue
				} else if err != nil {
					return nil, err
				}

				// Found a Srclibtoolchain file.
				path = filepath.Clean(path)

				var toolchainPath string
				if orig, present := origDirs[dir]; present {
					toolchainPath, _ = filepath.Rel(dir, path)
					if toolchainPath == "." {
						toolchainPath = ""
					}
					toolchainPath = orig + toolchainPath
				} else {
					toolchainPath, _ = filepath.Rel(dir, path)
				}
				toolchainPath = filepath.ToSlash(toolchainPath)
				

				if otherDir, seen := seen[toolchainPath]; seen {
					return nil, fmt.Errorf("saw 2 toolchains at path %s in dirs %s and %s", toolchainPath, otherDir, path)
				}
				seen[toolchainPath] = path

				info, err := newInfo(toolchainPath, path, ConfigFilename)
				if err != nil {
					return nil, err
				}
				found = append(found, info)

				// Disallow nested toolchains to speed up List. This
				// means that if DIR/Srclibtoolchain exists, no other
				// Srclibtoolchain files underneath DIR will be read.
				w.SkipDir()
			}
		}
	}
	return found, nil
}

func newInfo(toolchainPath, dir, configFile string) (*Info, error) {
	dockerfile := "Dockerfile"
	if _, err := os.Stat(filepath.Join(dir, dockerfile)); os.IsNotExist(err) {
		dockerfile = ""
	} else if err != nil {
		return nil, err
	}

	prog := filepath.Join(".bin", filepath.Base(toolchainPath))

	if runtime.GOOS == "windows" {
		prog = winExe(dir, prog)
	}

	if runtime.GOOS != "windows" {
		fi, err := os.Stat(filepath.Join(dir, prog))
		if os.IsNotExist(err) {
			prog = ""
		} else {
			if fi.Mode().Perm()&0111 == 0 {
				return nil, fmt.Errorf("installed toolchain program %q is not executable (+x)", prog)
			}
		}
	}

	return &Info{
		Path:       toolchainPath,
		Dir:        dir,
		ConfigFile: configFile,
		Program:    prog,
		Dockerfile: dockerfile,
	}, nil
}

// lookInPaths returns all files in paths (a colon-separated list of
// directories) matching the glob pattern.
func lookInPaths(pattern string, paths string) ([]string, error) {
	var found []string
	seen := map[string]struct{}{}
	for _, dir := range filepath.SplitList(paths) {
		if dir == "" {
			dir = "."
		}
		matches, err := filepath.Glob(filepath.Join(dir, pattern))
		if err != nil {
			return nil, err
		}
		for _, m := range matches {
			if _, seen := seen[m]; seen {
				continue
			}
			seen[m] = struct{}{}
			found = append(found, m)
		}
	}
	sort.Strings(found)
	return found, nil
}

// searches for matching Windows executable (.exe, .bat, .cmd)
func winExe(dir string, program string) (string) {
	candidate := program + ".exe"
	if _, err := os.Stat(filepath.Join(dir, candidate)); err == nil {
		return candidate
	}
	candidate = program + ".bat"
	if _, err := os.Stat(filepath.Join(dir, candidate)); err == nil {
		return candidate
	}
	candidate = program + ".cmd"
	if _, err := os.Stat(filepath.Join(dir, candidate)); err == nil {
		return candidate
	}
	return ""
}
