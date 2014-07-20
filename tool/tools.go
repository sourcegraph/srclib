package tool

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/kr/fs"
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

// A Tool is either a local executable program or a Docker container that wraps
// such a program. Tools perform tasks related to source analysis.
type Tool interface {
	// Name is the name or URI of the tool that it was originally referred to by
	// in Lookup.
	Name() string

	// Build prepares the tool, if needed. For example, for Dockerized tools, it
	// builds the Docker image.
	Build() error

	// Command returns an *exec.Cmd that will execute this tool, where dir is
	// the working directory of the command.
	//
	// Do not modify the returned Cmd's Dir field; some implementations of Tool
	// use dir to construct other parts of the Cmd, so it's important that all
	// references to the working directory are consistent.
	Command(dir string) (*exec.Cmd, error)

	// Operations lists the subcommands that this tool implements, such as
	// "scan".
	Operations() ([]string, error)

	// Type is either "installed program" or "Docker container"
	Type() string
}

// A ProgramTool is a local executable program tool that has been installed in
// the PATH.
type ProgramTool struct {
	// name of tool
	name string

	// Program (executable) path
	Program string
}

func (t *ProgramTool) Name() string { return t.name }
func (t *ProgramTool) Type() string { return "installed program" }

func (t *ProgramTool) Build() error { return nil }

func (t *ProgramTool) Command(dir string) (*exec.Cmd, error) {
	cmd := exec.Command(t.Program)
	cmd.Dir = dir
	return cmd, nil
}
func (t *ProgramTool) Operations() ([]string, error) {
	cmd, err := t.Command("")
	if err != nil {
		return nil, err
	}
	cmd.Args = append(cmd.Args, "help", "-q")
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return strings.Split(strings.TrimSpace(string(out)), "\n"), nil
}

// DockerTool is a Docker container that wraps a program.
type DockerTool struct {
	// name of tool
	name string

	// Dir containing Dockerfile
	Dir string

	// Dockerfile path
	Dockerfile string

	// ImageName of the Docker image
	ImageName string

	// built is whether Build() has completed successfully.
	built bool
}

func (t *DockerTool) Name() string { return t.name }
func (t *DockerTool) Type() string { return "Docker container" }

func (t *DockerTool) Build() error {
	t.ImageName = strings.Replace(t.name, "/", "-", -1)

	cmd := exec.Command("docker", "build", "-t", t.ImageName, ".")
	cmd.Dir = t.Dir
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s (command was: %v)", err, cmd.Args)
	}
	t.built = true
	return nil
}

func (t *DockerTool) Command(dir string) (*exec.Cmd, error) {
	if !t.built {
		if err := t.Build(); err != nil {
			return nil, err
		}
	}
	cmd := exec.Command("docker", "run", "--volume="+dir+":/src:ro", t.ImageName)
	cmd.Dir = dir
	return cmd, nil
}

func (t *DockerTool) Operations() ([]string, error) {
	tf := filepath.Join(t.Dir, "Srclibtool")
	data, err := ioutil.ReadFile(tf)
	if err != nil {
		return nil, err
	}

	var ops []string
	for _, line := range bytes.Split(data, []byte("\n")) {
		if bytes.HasPrefix(line, []byte("OP ")) {
			ops = append(ops, string(bytes.TrimSpace(line[len("OP "):])))
		}
	}
	return ops, nil
}

// Location is a place where tools can be stored (for example, in the PATH or
// the SRCLIBPATH).
type Location interface {
	Lookup(name string) (Tool, error)
	List() ([]Tool, error)
}

var (
	InstalledTools  = installedTools{}
	SrclibPathTools = srclibPathTools{}
)

type installedTools struct{}

// Lookup finds the named tool (adding a `src-tool-` prefix to the name if not
// already present) in the PATH.
func (l *installedTools) Lookup(name string) (Tool, error) {
	if !strings.HasPrefix(name, "src-tool-") {
		name = "src-tool-" + name
	}
	prog, err := exec.LookPath(name)
	if err != nil {
		return nil, err
	}
	return &ProgramTool{
		name:    name,
		Program: prog,
	}, nil
}

// List finds all installed tools in the PATH whose names match `src-tool-*` and
// returns their full paths.
func (l *installedTools) List() ([]Tool, error) {
	matches, err := lookInPaths("src-tool-*", os.Getenv("PATH"))
	if err != nil {
		return nil, err
	}

	var tools []Tool

	for _, m := range matches {
		fi, err := os.Stat(m)
		if err != nil {
			return nil, err
		}
		// executables only
		if fi.Mode().Perm()&0111 > 0 {
			tools = append(tools, &ProgramTool{name: strings.TrimPrefix(filepath.Base(m), "src-tool-"), Program: m})
		}
	}
	return tools, nil
}

type srclibPathTools struct{}

// Lookup finds the named tool in the SRCLIBPATH.
func (l *srclibPathTools) Lookup(name string) (Tool, error) {
	matches, err := lookInPaths(name, SrclibPath)
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("no tool %q found in SRCLIBPATH %q", name, SrclibPath)
	}

	dir := matches[0]
	df := filepath.Join(dir, "Dockerfile")
	if _, err := os.Stat(df); err != nil {
		return nil, err
	}
	return &DockerTool{
		name:       name,
		Dir:        dir,
		Dockerfile: df,
	}, nil
}

// List implements Location.
func (l *srclibPathTools) List() ([]Tool, error) {
	var found []Tool
	seen := map[string]struct{}{}

	dirs := strings.Split(SrclibPath, ":")

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
				// traverse symlinks but refer to symlinked trees' tools using
				// the path to them through the original entry in SrclibPath
				dirs = append(dirs, path+"/")
				origDirs[path+"/"] = dir
			} else if fi.Mode().IsRegular() && strings.ToLower(name) == "srclibtool" {
				var base string
				if orig, present := origDirs[dir]; present {
					base = orig
				} else {
					base = dir
				}
				name, _ := filepath.Rel(base, filepath.Dir(path))
				if _, seen := seen[name]; seen {
					continue
				}
				seen[name] = struct{}{}
				toolDir := filepath.Dir(path)
				found = append(found, &DockerTool{name: name, Dir: toolDir, Dockerfile: filepath.Join(toolDir, "Dockerfile")})
			}
		}
	}
	return found, nil
}

// Lookup finds the tool program by name.
//
// The search order is PATH (for programs named `src-tool-$name`) then
// SRCLIBPATH.
func Lookup(name string) (Tool, error) {
	tool, err := InstalledTools.Lookup(name)
	if err != nil {
		if err, ok := err.(*exec.Error); !ok || !os.IsNotExist(err.Err) {
			return nil, err
		}
	}
	if tool != nil {
		return tool, nil
	}

	return SrclibPathTools.Lookup(name)
}

// List finds all tools in the PATH and SRCLIBPATH.
func List() ([]Tool, error) {
	tools1, err := InstalledTools.List()
	if err != nil {
		return nil, err
	}

	tools2, err := SrclibPathTools.List()
	if err != nil {
		return nil, err
	}

	return append(tools1, tools2...), nil
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

// CommonOps is a list of ops (subcommands) that all tools must implement.
var CommonOps = map[string]struct{}{
	"version": struct{}{},
	"help":    struct{}{},
	"info":    struct{}{},
}
