package toolchain

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
)

var SrclibPath string

func init() {
	SrclibPath = os.Getenv("SRCLIBPATH")
	if SrclibPath == "" {
		user, err := user.Current()
		if err != nil {
			log.Fatal(err)
		}
		if user.HomeDir == "" {
			log.Fatal("Fatal: No SRCLIBPATH and current user %q has no home directory.", user.Username)
		}
		SrclibPath = filepath.Join(user.HomeDir, ".srclib")
	}
}

// LookupInSRCLIBPATH finds the named tool in the SRCLIBPATH.
//
// TODO(sqs): make this look up tools in the PATH, and add a param for setting
// whether direct or dockerized tools are preferred.
func LookupInSRCLIBPATH(name string) (string, error) {
	matches, err := lookInPaths(name, SrclibPath)
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("no tool %q found in SRCLIBPATH %q", name, SrclibPath)
	}
	return matches[0], nil
}

func LookupInPATH(name string) (string, error) {
	if !strings.HasPrefix(name, "src-tool-") {
		name = "src-tool-" + name
	}
	return exec.LookPath(name)
}

// FindAllInPATH finds all programs in the PATH whose names match `src-tool-*`
// and returns their full paths.
func FindAllInPATH() ([]string, error) {
	matches, err := lookInPaths("src-tool-*", os.Getenv("PATH"))
	if err != nil {
		return nil, err
	}

	// executables only
	var exes []string
	for _, m := range matches {
		fi, err := os.Stat(m)
		if err != nil {
			return nil, err
		}
		if fi.Mode().Perm()&0111 > 0 {
			exes = append(exes, m)
		}
	}
	return exes, nil
}

// lookInPaths returns all executables in paths (a colon-separated
// list of directories) matching the glob pattern.
func lookInPaths(pattern string, paths string) ([]string, error) {
	var found []string
	seen := map[string]struct{}{}
	for _, dir := range strings.Split(paths, ":") {
		if dir == "" {
			dir = "."
		}
		matches, err := filepath.Glob(dir + "/" + pattern)
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
	return found, nil
}
