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
	unit.Register("python", &fauxPackage{})
}

type fauxPackage struct{}

func (p *fauxPackage) Name() string {
	return "python-faux-package"
}

func (p *fauxPackage) RootDir() string {
	return "."
}

func (p *fauxPackage) Paths() []string {
	return nil
}

type fauxScanner struct{}

func (p *fauxScanner) Scan(dir string, c *config.Repository, x *task2.Context) ([]unit.SourceUnit, error) {
	isPython := false
	walker := fs.Walk(dir)
	for walker.Step() {
		if !walker.Stat().IsDir() && filepath.Ext(walker.Path()) == ".py" {
			isPython = true
			break
		}
	}

	if isPython {
		return []unit.SourceUnit{&fauxPackage{}}, nil
	} else {
		return nil, nil
	}
}
