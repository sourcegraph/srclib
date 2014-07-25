package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/sourcegraph/srclib/repo"
	"github.com/sourcegraph/srclib/toolchain"
	"github.com/sourcegraph/srclib/unit"
)

var Filename = "Srcfile"

var (
	ErrInvalidFilePath = errors.New("invalid file path specified in config (above config root dir or source unit dir)")
)

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
	SourceUnits []*unit.SourceUnit `json:",omitempty"`

	// Scanners to use when scanning this tree for source units. If not set,
	// this defaults to DefaultScanners.
	Scanners []*toolchain.ToolRef

	Tools map[string][]string

	// Config is a map from unit spec (i.e., UnitType:UnitName) to an arbitrary
	// property map. It is used to pass extra configuration settings to all of
	// the handlers for matching source units.
	Config map[string]map[string]string
}

// DefaultScanners are the scanners used for a Tree if none are manually
// specified in a Srcfile.
var DefaultScanners = []*toolchain.ToolRef{
	{"github.com/sourcegraph/srclib-go", "scan"},
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

// ParseRepository parses and validates the JSON representation of a
// repository's configuration. If the JSON representation is empty
// (len(configJSON) == 0), it returns the default configuration for the
// repository.
func ParseRepository(configJSON []byte, repoURI repo.URI) (*Repository, error) {
	var c *Repository
	if len(configJSON) > 0 {
		err := json.Unmarshal(configJSON, &c)
		if err != nil {
			return nil, err
		}
	} else {
		c = new(Repository)
	}

	return c.finish(repoURI)
}

func (c *Repository) finish(repoURI repo.URI) (*Repository, error) {
	err := c.validate()
	if err != nil {
		return nil, err
	}
	c.URI = repoURI
	if c.Scanners == nil {
		c.Scanners = DefaultScanners
	}
	return c, nil
}
