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
	scan.Register("python", &pythonScanner{})
	unit.Register("python", &pythonPackage{})
}

type pythonPackage struct {
	name string
}

func (p *pythonPackage) Name() string {
	return p.name
}

func (p *pythonPackage) RootDir() string {
	return "."
}

func (p *pythonPackage) Paths() []string {
	return nil
}

type pythonScanner struct{}

func (p *pythonScanner) Scan(dir string, c *config.Repository, x *task2.Context) ([]unit.SourceUnit, error) {
	isPython := false
	walker := fs.Walk(dir)
	for walker.Step() {
		if !walker.Stat().IsDir() && filepath.Ext(walker.Path()) == ".py" {
			isPython = true
			break
		}
	}

	if isPython {
		return []unit.SourceUnit{&pythonPackage{filepath.Base(dir)}}, nil
	} else {
		return nil, nil
	}
}
