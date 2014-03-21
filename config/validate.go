package config

import (
	"path/filepath"
	"strings"
)

func (c *Repository) validate() error {
	for _, u := range c.SourceUnits {
		paths := u.Paths()
		for _, p := range paths {
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
