package container

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
)

// Runner runs commands specified by a Command. The default implementation uses
// Docker, but MockExecutor also implements this interface and may be used
// during testing (by temporarily changing the DefaultRunner variable). The sole
// purpose of this interface is to allow mocking.
type Runner interface {
	Run(*Command) ([]byte, error)
}

var DefaultRunner Runner = dockerRunner{}

type dockerRunner struct{}

func (_ dockerRunner) Run(c *Command) ([]byte, error) {
	tmpDir, err := ioutil.TempDir("", "sg-docker")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	image := "sg-temp:" + filepath.Base(tmpDir)

	cmdJSON, err := json.Marshal(c.Cmd)
	if err != nil {
		return nil, err
	}

	// Build Docker container.
	dockerfile := c.Dockerfile

	// AddFiles
	for i, f := range c.AddFiles {
		if i == 0 {
			dockerfile = append(dockerfile, '\n')
		}

		// Copy to the Docker context dir so we can ADD it.
		name := filepath.Base(f[0])
		err = cp(f[0], filepath.Join(tmpDir, name))
		if err != nil {
			return nil, err
		}
		dockerfile = append(dockerfile, []byte(fmt.Sprintf("ADD %s %s\n", "./"+name, f[1]))...)
	}

	// AddDirs
	for i, d := range c.AddDirs {
		if i == 0 {
			dockerfile = append(dockerfile, '\n')
		}

		// Move repository to Docker context dir and ADD it.
		dirName := filepath.Base(d[0])
		err := cp(d[0], filepath.Join(tmpDir, dirName))
		if err != nil {
			return nil, err
		}
		dockerfile = append(dockerfile, []byte(fmt.Sprintf("ADD %s %s\n", "./"+dirName, d[1]))...)
	}

	dockerfile = append(dockerfile, c.PreCmdDockerfile...)

	dockerfile = append(dockerfile, []byte(fmt.Sprintf("\nCMD %s", cmdJSON))...)
	err = ioutil.WriteFile(filepath.Join(tmpDir, "Dockerfile"), []byte(dockerfile), 0700)
	if err != nil {
		return nil, err
	}
	buildCmd := exec.Command("docker", "build", "--rm=false", "-t", image, ".")
	buildCmd.Dir = tmpDir
	buildCmd.Stdout, buildCmd.Stderr = c.Stdout, c.Stderr
	err = buildCmd.Run()
	if err != nil {
		return nil, err
	}

	runOptions := append([]string{}, c.RunOptions...)
	if c.Dir != "" {
		runOptions = append(runOptions, "--workdir="+c.Dir)
	}
	args := append([]string{"run"}, runOptions...)
	args = append(args, image)
	runCmd := exec.Command("docker", args...)
	runCmd.Stderr = os.Stderr

	// Get original output.
	data, err := runCmd.Output()
	if err != nil {
		return nil, err
	}

	// Transform.
	if c.Transform != nil {
		data, err = c.Transform(data)
		if err != nil {
			return nil, err
		}
	}

	return data, nil
}

type MockRunner struct {
	Cmds []*Command
}

func (r *MockRunner) Run(c *Command) ([]byte, error) {
	r.Cmds = append(r.Cmds, c)
	return nil, nil
}

var _ Runner = &MockRunner{}

func cp(src, dst string) error {
	cmd := exec.Command("cp", "-R", "--no-dereference", "--preserve=mode,ownership,timestamps", src, dst)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("cp %s %s failed: %s: %q", src, dst, err, out)
	}
	return nil
}
