package golang

import "sourcegraph.com/sourcegraph/srcgraph/config"

func init() {
	config.Register("go", &Config{})
}

type Config struct {
	// BaseImportPath is prepended to the dirs of all GoPackage source units to
	// yield the import path of the package. E.g., if BaseImportPath is
	// "github.com/foo/bar", then a GoPackage with dir "." will be at import
	// path "github.com/foo/bar" and a GoPackage with dir "qux/baz" will be at
	// import path "github.com/foo/bar/qux/baz".
	BaseImportPath string
}

func (v *goVersion) goConfig(c *config.Repository) *Config {
	goConfig, _ := c.Global["go"].(*Config)
	if goConfig == nil {
		goConfig = new(Config)
	}
	if goConfig.BaseImportPath == "" {
		goConfig.BaseImportPath = string(c.URI)
	}
	return goConfig
}
