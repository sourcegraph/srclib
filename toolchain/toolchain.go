package toolchain

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"

	"sourcegraph.com/sourcegraph/srclib"
)

// Dir returns the directory where the named toolchain lives (under
// the SRCLIBPATH). If the toolchain already exists in any of the
// entries of SRCLIBPATH, that directory is returned. Otherwise a
// nonexistent directory in the first SRCLIBPATH entry is returned.
func Dir(toolchainPath string) (string, error) {
	toolchainPath = filepath.Clean(toolchainPath)

	dir, err := lookupToolchain(toolchainPath)
	if os.IsNotExist(err) {
		return filepath.Join(filepath.SplitList(srclib.Path)[0], toolchainPath), nil
	}
	if err != nil {
		err = &os.PathError{Op: "toolchain.Dir", Path: toolchainPath, Err: err}
	}
	return dir, err
}

// Info describes a toolchain.
type Info struct {
	// Path is the toolchain's path (not a directory path) underneath the
	// SRCLIBPATH. It consists of the URI of this repository's toolchain plus
	// its subdirectory path within the repository. E.g., "github.com/foo/bar"
	// for a toolchain defined in the root directory of that repository.
	Path string

	// Dir is the filesystem directory that defines this toolchain.
	Dir string

	// ConfigFile is the path to the Srclibtoolchain file, relative to Dir.
	ConfigFile string

	// Program is the path to the executable program (relative to Dir) to run to
	// invoke this toolchain.
	Program string `json:",omitempty"`
}

// ReadConfig reads and parses the Srclibtoolchain config file for the
// toolchain.
func (t *Info) ReadConfig() (*Config, error) {
	f, err := os.Open(filepath.Join(t.Dir, t.ConfigFile))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var c *Config
	if err := json.NewDecoder(f).Decode(&c); err != nil {
		return nil, err
	}
	return c, nil
}

// Open opens a toolchain by path.
func Open(path string) (Toolchain, error) {
	tc, err := Lookup(path)
	if err != nil {
		return nil, err
	}

	if tc.Program != "" {
		return &programToolchain{filepath.Join(tc.Dir, tc.Program)}, nil
	}
	return nil, &os.PathError{Op: "toolchain.Open", Path: path, Err: os.ErrNotExist}
}

// A Toolchain is an executable program. Toolchains contain tools (as
// subcommands), which perform actions or analysis on a project's
// source code.
type Toolchain interface {
	// Command returns an *exec.Cmd that will execute this toolchain. Do not use
	// this to execute a tool in this toolchain; use OpenTool instead.
	//
	// Do not modify the returned Cmd's Dir field; some implementations of
	// Toolchain use dir to construct other parts of the Cmd, so it's important
	// that all references to the working directory are consistent.
	Command() (*exec.Cmd, error)

	// Build prepares the toolchain, if needed.
	Build() error

	// IsBuilt returns whether the toolchain is built and can be executed (using
	// Command).
	IsBuilt() (bool, error)
}

// A programToolchain is a local executable program toolchain that has been installed in
// the PATH.
type programToolchain struct {
	// program (executable) path
	program string
}

// IsBuilt always returns true for programs.
func (t *programToolchain) IsBuilt() (bool, error) { return true, nil }

// Build is a no-op for programs.
func (t *programToolchain) Build() error { return nil }

// Command returns an *exec.Cmd that executes this program.
func (t *programToolchain) Command() (*exec.Cmd, error) {
	cmd := exec.Command(t.program)
	return cmd, nil
}
