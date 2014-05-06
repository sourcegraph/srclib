package python

import (
	"path/filepath"

	"github.com/kr/fs"
	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/scan"
	"sourcegraph.com/sourcegraph/srcgraph/task2"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
)

func init() {
	scan.Register("python", &fauxScanner{})
	unit.Register("python", &FauxPackage{})
}

type fauxScanner struct{}

func (p *fauxScanner) Scan(dir string, c *config.Repository, x *task2.Context) ([]unit.SourceUnit, error) {
	var files []string
	walker := fs.Walk(dir)
	for walker.Step() {
		if err := walker.Err(); err == nil && !walker.Stat().IsDir() && filepath.Ext(walker.Path()) == ".py" {
			file, _ := filepath.Rel(dir, walker.Path())
			files = append(files, file)
		}
	}

	if len(files) > 0 {
		return []unit.SourceUnit{&FauxPackage{Files: files}}, nil
	} else {
		return nil, nil
	}
}
