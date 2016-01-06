package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"strings"

	"sourcegraph.com/sourcegraph/srclib"
	"sourcegraph.com/sourcegraph/srclib/graph2"
)

// Tree2 represents the config for an entire source tree.
type Tree2 struct {
	// Units is a list of build units in the tree, either specified
	// manually in the Srcfile or discovered automatically by the scanner.
	Units []*graph2.Unit `json:",omitempty"`

	// Scanners to use to scan for source units in this tree.
	Scanners []*srclib.ToolRef `json:",omitempty"`

	// SkipDirs is a list of directory trees that are skipped. That is, any
	// source units (produced by scanners) whose Dir is in a skipped dir tree is
	// not processed further.
	SkipDirs []string `json:",omitempty"`

	// SkipUnits is a list of source units that are skipped. That is,
	// any scanned source units whose name and type exactly matches a
	// name and type pair in SkipUnits is skipped.
	SkipUnits []struct{ Name, Type string } `json:",omitempty"`

	// TODO(sqs): Add some type of field that lets the Srcfile and the scanners
	// have input into which tools get used during the execution phase. Right
	// now, we're going to try just using the system defaults (srclib-*) and
	// then add more flexibility when we are more familiar with the system.

	// Config is an arbitrary key-value property map. Properties are copied
	// verbatim to each source unit that is scanned in this tree.
	Config map[string]interface{} `json:",omitempty"`
}

// ReadTreeConfig parses and validates the configuration for a source
// tree. If no Srcfile exists, it returns the default configuration
// for the source tree. If an overridden configuration is specified
// for the source tree (hard-coded in the Go code), then it is used
// instead of the Srcfile or the default configuration.
func ReadTreeConfig(dir string) (*Tree2, error) {
	var c *Tree2
	if f, err := os.Open(filepath.Join(dir, Filename)); err == nil {
		defer f.Close()
		err = json.NewDecoder(f).Decode(&c)
		if err != nil {
			return nil, err
		}
	} else if os.IsNotExist(err) {
		err = nil
		c = new(Tree2)
	} else {
		return nil, err
	}

	if err := c.validate(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Tree2) validate() error {
	for _, u := range c.Units {
		for _, p := range u.Files {
			p = filepath.Clean(p)
			if filepath.IsAbs(p) {
				return ErrInvalidFilePath
			}
			if p == ".." || strings.HasPrefix(p, ".."+string(filepath.Separator)) {
				return ErrInvalidFilePath
			}
		}
	}
	return nil
}
