package config

import (
	"path/filepath"
	"strings"
)

func (c *Tree) validate() error {
	for _, u := range c.SourceUnits {
		for _, p := range u.Files {
			p = filepath.Clean(p)
			if filepath.IsAbs(p) {
				return ErrInvalidFilePath
			}
			if p == ".." || strings.HasPrefix(p, "../") {
				return ErrInvalidFilePath
			}
		}
	}
	return nil
}
