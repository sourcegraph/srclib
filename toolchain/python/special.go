package python

import (
	"github.com/sourcegraph/srclib/dep2"
	"github.com/sourcegraph/srclib/repo"
	"github.com/sourcegraph/srclib/unit"
)

type repoUnit struct {
	Repo     repo.URI
	Unit     string
	UnitType string
}

// Special cases

var hardcodedScan = map[repo.URI][]unit.SourceUnit{
	stdLibRepo: []unit.SourceUnit{stdLibUnit},
	extensionsTestRepo: []unit.SourceUnit{extensionsTestUnit},
}

var hardcodedDep = map[repoUnit][]*dep2.RawDependency{
	repoUnit{stdLibRepo, stdLibUnit.Name(), DistPackageDisplayName}: nil,
	repoUnit{extensionsTestRepo, extensionsTestUnit.Name(), DistPackageDisplayName}: nil,
}
