package scan

import (
	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/container"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
)

// ScannerBuilder implementations define a Docker container that, when run,
// scans for source units in a repository.
type ScannerBuilder interface {
	// Scan returns a command that will search files in dir for source units,
	// using the supplied repository configuration. The dir refers to a
	// directory on the host (not the container). Typically, the container
	// mounts this host dir.
	BuildScanner(dir string, c *config.Repository) (*container.Command, error)

	// UnmarshalSourceUnits unmarshals data, which is the stdout output from a
	// command returned by Scan, and returns a list of source units that data
	// represents. For example, if the command outputs JSON, then
	// UnmarshalSourceUnits will typically unmarshal the JSON to the correct
	// implementation of unit.SourceUnit.
	//
	// It's necessary to define UnmarshalSourceUnits because only the scanner
	// knows which implementation of unit.SourceUnit to use.
	UnmarshalSourceUnits(data []byte) ([]unit.SourceUnit, error)
}

type DockerScanner struct {
	ScannerBuilder
}

func (s DockerScanner) Scan(dir string, c *config.Repository) ([]unit.SourceUnit, error) {
	cmd, err := s.BuildScanner(dir, c)
	if err != nil {
		return nil, err
	}

	data, err := cmd.Run()
	if err != nil {
		return nil, err
	}

	units, err := s.UnmarshalSourceUnits(data)
	if err != nil {
		return nil, err
	}

	return units, nil
}
