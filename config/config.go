package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/sourcegraph/srclib/repo"
	"github.com/sourcegraph/srclib/unit"
)

// Filename is the name of the file that configures a directory tree or
// repository. It is intended to be used by repository authors.
var Filename = "Srcfile"

// Repository represents the config for an entire repository.
type Repository struct {
	// URI is the repository's clone URI.
	URI repo.URI `json:",omitempty"`

	// Tree is the configuration for the top-level directory tree in the
	// repository.
	Tree
}

// Tree represents the config for a directory and its subdirectories.
type Tree struct {
	// SourceUnits is a list of source units in the repository, either specified
	// manually in the Srcfile or discovered automatically by the scanner.
	SourceUnits []*unit.SourceUnit `json:",omitempty"`

	// TODO(sqs): Add some type of field that lets the Srcfile and the scanners
	// have input into which tools get used during the execution phase. Right
	// now, we're going to try just using the system defaults (srclib-*) and
	// then add more flexibility when we are more familiar with the system.
}

// ReadRepository parses and validates the configuration for a repository. If no
// Srcfile exists, it returns the default configuration for the repository. If
// an overridden configuration is specified for the repository (hard-coded in
// the Go code), then it is used instead of the Srcfile or the default
// configuration.
func ReadRepository(dir string, repoURI repo.URI) (*Repository, error) {
	var c *Repository
	if oc, overridden := overrides[repoURI]; overridden {
		c = oc
	} else if f, err := os.Open(filepath.Join(dir, Filename)); err == nil {
		defer f.Close()
		err = json.NewDecoder(f).Decode(&c)
		if err != nil {
			return nil, err
		}
	} else if os.IsNotExist(err) {
		err = nil
		c = new(Repository)
	} else {
		return nil, err
	}

	return c.finish(repoURI)
}

func (c *Repository) finish(repoURI repo.URI) (*Repository, error) {
	err := c.validate()
	if err != nil {
		return nil, err
	}
	c.URI = repoURI
	return c, nil
}
