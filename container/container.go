package container

import "io"

type Container struct {
	Dockerfile []byte

	PreCmdDockerfile []byte

	// RunOptions are supplied to the `docker run` subcommand.
	RunOptions []string

	// Cmd, if non-empty, is added to the Dockerfile as a "CMD ..." directive.
	Cmd []string

	// AddFiles is a list of files to add to the Docker container's filesystem
	// (using "ADD"). The 1st element of the array is the host file and the 2nd
	// element is the destination path inside the container.
	AddFiles [][2]string

	// AddDirs is a list of dirs to add (recursively) to the Docker container's
	// dirsystem (using "ADD"). The 1st element of the array is the host dir and
	// the 2nd element is the destination path inside the container.
	AddDirs [][2]string

	// Stderr is the io.Writer to write error and log output to.
	Stderr io.Writer

	// Stdout is the io.Writer to write output to.
	Stdout io.Writer

	// Dir is the container wd ("WORKDIR") to run Docker container commands in.
	Dir string
}

type Command struct {
	Container

	Transform func(orig []byte) ([]byte, error)
}

func (c *Command) Run() ([]byte, error) {
	return DefaultRunner.Run(c)
}
