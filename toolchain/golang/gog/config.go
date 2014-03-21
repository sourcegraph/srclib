package gog

import (
	"go/build"

	"code.google.com/p/go.tools/go/loader"
	"code.google.com/p/go.tools/go/types"
)

var Default = loader.Config{
	TypeChecker:     types.Config{FakeImportC: true},
	Build:           &build.Default,
	AllowTypeErrors: true,
}
