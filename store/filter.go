package store

import (
	"log"
	"strings"

	"sourcegraph.com/sourcegraph/srclib/graph"
	"sourcegraph.com/sourcegraph/srclib/unit"
)

// A DefFilter filters a set of defs to only those for which SelectDef
// returns true.
type DefFilter interface {
	SelectDef(*graph.Def) bool
}

// A DefFilterFunc is a DefFilter that selects only those defs for
// which the func returns true.
type DefFilterFunc func(*graph.Def) bool

// SelectDef calls f(def).
func (f DefFilterFunc) SelectDef(def *graph.Def) bool { return f(def) }

var allDefs = DefFilterFunc(func(*graph.Def) bool { return true })

func defPathFilter(path string) DefFilter {
	return DefFilterFunc(func(def *graph.Def) bool { return def.Path == path })
}

// A RefFilter filters a set of refs to only those for which SelectRef
// returns true.
type RefFilter interface {
	SelectRef(*graph.Ref) bool
}

// A RefFilterFunc is a RefFilter that selects only those refs for
// which the func returns true.
type RefFilterFunc func(*graph.Ref) bool

// SelectRef calls f(ref).
func (f RefFilterFunc) SelectRef(ref *graph.Ref) bool { return f(ref) }

var allRefs = RefFilterFunc(func(*graph.Ref) bool { return true })

// A UnitFilter filters a set of units to only those for which Select
// returns true.
type UnitFilter interface {
	SelectUnit(*unit.SourceUnit) bool
}

// A UnitFilterFunc is a UnitFilter that selects only those units for
// which the func returns true.
type UnitFilterFunc func(*unit.SourceUnit) bool

// SelectUnit calls f(unit).
func (f UnitFilterFunc) SelectUnit(unit *unit.SourceUnit) bool { return f(unit) }

var allUnits = UnitFilterFunc(func(*unit.SourceUnit) bool { return true })

// A VersionFilter filters a set of versions to only those for which SelectVersion
// returns true.
type VersionFilter interface {
	SelectVersion(*Version) bool
}

// A VersionFilterFunc is a VersionFilter that selects only those
// versions for which the func returns true.
type VersionFilterFunc func(*Version) bool

// SelectVersion calls f(version).
func (f VersionFilterFunc) SelectVersion(version *Version) bool { return f(version) }

var allVersions = VersionFilterFunc(func(*Version) bool { return true })

// A RepoFilter filters a set of repos to only those for which SelectRepo
// returns true.
type RepoFilter interface {
	SelectRepo(string) bool
}

// A RepoFilterFunc is a RepoFilter that selects only those repos for
// which the func returns true.
type RepoFilterFunc func(string) bool

// SelectRepo calls f(repo).
func (f RepoFilterFunc) SelectRepo(repo string) bool { return f(repo) }

var allRepos = RepoFilterFunc(func(string) bool { return true })

// ByUnitFilter is implemented by filters that restrict their
// selections to items from a specific source unit. It allows the
// store to optimize calls by skipping data that it knows is not in
// the specified source unit.
type ByUnitFilter interface {
	ByUnitType() string
	ByUnit() string
}

// ByUnit creates a new filter by source unit name and type. It panics
// if either unit or unitType is empty.
func ByUnit(unitType, unit string) interface {
	DefFilter
	RefFilter
	UnitFilter
	ByUnitFilter
} {
	if unit == "" {
		panic("unit: empty")
	}
	if unitType == "" {
		panic("unitType: empty")
	}
	if strings.Contains(unitType, "/") {
		log.Printf("WARNING: srclib store.ByUnit was called with a source unit type of %q, which resembles a unit *name*. Did you mix up the order of ByUnit's arguments?", unitType)
	}
	return byUnitFilter{unitType, unit}
}

type byUnitFilter struct{ unitType, unit string }

func (f byUnitFilter) ByUnitType() string { return f.unitType }
func (f byUnitFilter) ByUnit() string     { return f.unit }
func (f byUnitFilter) SelectDef(def *graph.Def) bool {
	return def.Unit == f.unit && def.UnitType == f.unitType
}
func (f byUnitFilter) SelectRef(ref *graph.Ref) bool {
	return ref.Unit == f.unit && ref.UnitType == f.unitType
}
func (f byUnitFilter) SelectUnit(unit *unit.SourceUnit) bool {
	return unit.Name == f.unit && unit.Type == f.unitType
}

// ByCommitIDFilter is implemented by filters that restrict their
// selection to items at a specific commit ID. It allows the store to
// optimize calls by skipping data that it knows is not at the
// specified commit.
type ByCommitIDFilter interface {
	ByCommitID() string
}

// ByCommitID creates a new filter by commit ID. It panics if commitID
// is empty.
func ByCommitID(commitID string) interface {
	DefFilter
	RefFilter
	UnitFilter
	VersionFilter
	ByCommitIDFilter
} {
	if commitID == "" {
		panic("commitID: empty")
	}
	return byCommitIDFilter{commitID}
}

type byCommitIDFilter struct{ commitID string }

func (f byCommitIDFilter) ByCommitID() string { return f.commitID }
func (f byCommitIDFilter) SelectDef(def *graph.Def) bool {
	return def.CommitID == f.commitID
}
func (f byCommitIDFilter) SelectRef(ref *graph.Ref) bool {
	return ref.CommitID == f.commitID
}
func (f byCommitIDFilter) SelectUnit(unit *unit.SourceUnit) bool {
	return unit.CommitID == f.commitID
}
func (f byCommitIDFilter) SelectVersion(version *Version) bool {
	return version.CommitID == f.commitID
}

// ByRepoFilter is implemented by filters that restrict their
// selections to items in a specific repository. It allows the store
// to optimize calls by skipping data that it knows is not in the
// specified repository.
type ByRepoFilter interface {
	ByRepo() string
}

// ByRepo creates a new filter by repository. It panics if repo is
// empty.
func ByRepo(repo string) interface {
	DefFilter
	RefFilter
	UnitFilter
	VersionFilter
	RepoFilter
	ByRepoFilter
} {
	if repo == "" {
		panic("repo: empty")
	}
	return byRepoFilter{repo}
}

type byRepoFilter struct{ repo string }

func (f byRepoFilter) ByRepo() string { return f.repo }
func (f byRepoFilter) SelectDef(def *graph.Def) bool {
	return def.Repo == f.repo
}
func (f byRepoFilter) SelectRef(ref *graph.Ref) bool {
	return ref.Repo == f.repo
}
func (f byRepoFilter) SelectUnit(unit *unit.SourceUnit) bool {
	return unit.Repo == f.repo
}
func (f byRepoFilter) SelectVersion(version *Version) bool {
	return version.Repo == f.repo
}
func (f byRepoFilter) SelectRepo(repo string) bool {
	return repo == f.repo
}

// ByRepoAndCommitID returns a filter by both repo and commit ID (both
// must match for an item to be selected by this filter). It panics if
// either repo or commitID is empty.
func ByRepoAndCommitID(repo, commitID string) interface {
	ByRepoFilter
	ByCommitIDFilter
} {
	if repo == "" {
		panic("repo: empty")
	}
	if commitID == "" {
		panic("commitID: empty")
	}
	return byRepoAndCommitIDFilter{repo, commitID}
}

type byRepoAndCommitIDFilter struct{ repo, commitID string }

func (f byRepoAndCommitIDFilter) ByRepo() string     { return f.repo }
func (f byRepoAndCommitIDFilter) ByCommitID() string { return f.commitID }
func (f byRepoAndCommitIDFilter) SelectDef(def *graph.Def) bool {
	return (def.Repo == "" || def.Repo == f.repo) && (def.CommitID == "" || def.CommitID == f.commitID)
}
func (f byRepoAndCommitIDFilter) SelectRef(ref *graph.Ref) bool {
	return (ref.Repo == "" || ref.Repo == f.repo) && (ref.CommitID == "" || ref.CommitID == f.commitID)
}
func (f byRepoAndCommitIDFilter) SelectUnit(unit *unit.SourceUnit) bool {
	return (unit.Repo == "" || unit.Repo == f.repo) && (unit.CommitID == "" || unit.CommitID == f.commitID)
}
func (f byRepoAndCommitIDFilter) SelectVersion(version *Version) bool {
	return (version.Repo == "" || version.Repo == f.repo) && version.CommitID == f.commitID
}

// ByUnitKey returns a filter by a source unit key. It panics if any
// fields on the unit key are not set. To filter by only source unit
// name and type, use ByUnit.
func ByUnitKey(key unit.Key) interface {
	DefFilter
	RefFilter
	UnitFilter
	ByRepoFilter
	ByCommitIDFilter
	ByUnitFilter
} {
	if key.Repo == "" {
		panic("key.Repo: empty")
	}
	if key.CommitID == "" {
		panic("key.CommitID: empty")
	}
	if key.UnitType == "" {
		panic("key.UnitType: empty")
	}
	if key.Unit == "" {
		panic("key.Unit: empty")
	}
	return byUnitKeyFilter{key}
}

type byUnitKeyFilter struct{ key unit.Key }

func (f byUnitKeyFilter) ByRepo() string     { return f.key.Repo }
func (f byUnitKeyFilter) ByCommitID() string { return f.key.CommitID }
func (f byUnitKeyFilter) ByUnitType() string { return f.key.UnitType }
func (f byUnitKeyFilter) ByUnit() string     { return f.key.Unit }
func (f byUnitKeyFilter) SelectDef(def *graph.Def) bool {
	return (def.Repo == "" || def.Repo == f.key.Repo) && (def.CommitID == "" || def.CommitID == f.key.CommitID) &&
		(def.UnitType == "" || def.UnitType == f.key.UnitType) && (def.Unit == "" || def.Unit == f.key.Unit)
}
func (f byUnitKeyFilter) SelectRef(ref *graph.Ref) bool {
	return (ref.Repo == "" || ref.Repo == f.key.Repo) && (ref.CommitID == "" || ref.CommitID == f.key.CommitID) &&
		(ref.UnitType == "" || ref.UnitType == f.key.UnitType) && (ref.Unit == "" || ref.Unit == f.key.Unit)
}
func (f byUnitKeyFilter) SelectUnit(unit *unit.SourceUnit) bool {
	return (unit.Repo == "" || unit.Repo == f.key.Repo) && (unit.CommitID == "" || unit.CommitID == f.key.CommitID) &&
		(unit.Type == "" || unit.Type == f.key.UnitType) && (unit.Name == "" || unit.Name == f.key.Unit)
}

// ByDefKey returns a filter by a def key. It panics if any fields on
// the def key are not set.
func ByDefKey(key graph.DefKey) interface {
	DefFilter
	ByRepoFilter
	ByCommitIDFilter
	ByUnitFilter
} {
	if key.Repo == "" {
		panic("key.Repo: empty")
	}
	if key.CommitID == "" {
		panic("key.CommitID: empty")
	}
	if key.UnitType == "" {
		panic("key.UnitType: empty")
	}
	if key.Unit == "" {
		panic("key.Unit: empty")
	}
	if key.Path == "" {
		panic("key.Path: empty")
	}
	return byDefKeyFilter{key}
}

type byDefKeyFilter struct{ key graph.DefKey }

func (f byDefKeyFilter) ByRepo() string     { return f.key.Repo }
func (f byDefKeyFilter) ByCommitID() string { return f.key.CommitID }
func (f byDefKeyFilter) ByUnitType() string { return f.key.UnitType }
func (f byDefKeyFilter) ByUnit() string     { return f.key.Unit }
func (f byDefKeyFilter) ByDefPath() string  { return f.key.Path }
func (f byDefKeyFilter) SelectDef(def *graph.Def) bool {
	return (def.Repo == "" || def.Repo == f.key.Repo) && (def.CommitID == "" || def.CommitID == f.key.CommitID) &&
		(def.UnitType == "" || def.UnitType == f.key.UnitType) && (def.Unit == "" || def.Unit == f.key.Unit) &&
		def.Path == f.key.Path
}

// A storesFilter is a named type that only exists for cosmetic
// purposes. It is passed to the methods of repoStores, treeStores,
// and unitStores that returns the map of stores. If it is a known
// By*Filter type (e.g., ByRepoFilter), those methods use it to
// restrict the contents of the map they return. Otherwise the full
// map of stores is returned.
type storesFilter interface{}
