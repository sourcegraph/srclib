package config

import "sourcegraph.com/sourcegraph/srclib/repo"

// TODO(sqs): remove this when we reenable overrides.go

var overrides = map[repo.URI]*Repository{}
