package config

import "github.com/sourcegraph/srclib/repo"

// TODO(sqs): remove this when we reenable overrides.go

var overrides = map[repo.URI]*Repository{}
