package python

import (
	"sourcegraph.com/sourcegraph/srcgraph/dep2"
	"sourcegraph.com/sourcegraph/srcgraph/repo"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
)

type repoUnit struct {
	Repo     repo.URI
	Unit     string
	UnitType string
}

// Special cases

var hardcodedScan = map[repo.URI][]unit.SourceUnit{
	stdLibRepo: []unit.SourceUnit{stdLibUnit},
}

var hardcodedDep = map[repoUnit][]*dep2.RawDependency{
	repoUnit{stdLibRepo, stdLibUnit.Name(), DistPackageDisplayName}: nil,
}
