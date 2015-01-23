package store

import (
	"log"
	"reflect"
	"strings"

	"sourcegraph.com/sourcegraph/srclib/graph"
	"sourcegraph.com/sourcegraph/srclib/unit"
)

// A DefFilter filters a set of defs to only those for which SelectDef
// returns true.
type DefFilter interface {
	SelectDef(*graph.Def) bool
}

type defFilters []DefFilter

func (fs defFilters) SelectDef(def *graph.Def) bool {
	for _, f := range fs {
		if !f.SelectDef(def) {
			return false
		}
	}
	return true
}

// A DefFilterFunc is a DefFilter that selects only those defs for
// which the func returns true.
type DefFilterFunc func(*graph.Def) bool

// SelectDef calls f(def).
func (f DefFilterFunc) SelectDef(def *graph.Def) bool { return f(def) }

func defPathFilter(path string) DefFilter {
	return DefFilterFunc(func(def *graph.Def) bool { return def.Path == path })
}

// A RefFilter filters a set of refs to only those for which SelectRef
// returns true.
type RefFilter interface {
	SelectRef(*graph.Ref) bool
}

type refFilters []RefFilter

func (fs refFilters) SelectRef(ref *graph.Ref) bool {
	for _, f := range fs {
		if !f.SelectRef(ref) {
			return false
		}
	}
	return true
}

// A RefFilterFunc is a RefFilter that selects only those refs for
// which the func returns true.
type RefFilterFunc func(*graph.Ref) bool

// SelectRef calls f(ref).
func (f RefFilterFunc) SelectRef(ref *graph.Ref) bool { return f(ref) }

// A UnitFilter filters a set of units to only those for which Select
// returns true.
type UnitFilter interface {
	SelectUnit(*unit.SourceUnit) bool
}

type unitFilters []UnitFilter

func (fs unitFilters) SelectUnit(unit *unit.SourceUnit) bool {
	for _, f := range fs {
		if !f.SelectUnit(unit) {
			return false
		}
	}
	return true
}

// A UnitFilterFunc is a UnitFilter that selects only those units for
// which the func returns true.
type UnitFilterFunc func(*unit.SourceUnit) bool

// SelectUnit calls f(unit).
func (f UnitFilterFunc) SelectUnit(unit *unit.SourceUnit) bool { return f(unit) }

// A VersionFilter filters a set of versions to only those for which SelectVersion
// returns true.
type VersionFilter interface {
	SelectVersion(*Version) bool
}

type versionFilters []VersionFilter

func (fs versionFilters) SelectVersion(version *Version) bool {
	for _, f := range fs {
		if !f.SelectVersion(version) {
			return false
		}
	}
	return true
}

// A VersionFilterFunc is a VersionFilter that selects only those
// versions for which the func returns true.
type VersionFilterFunc func(*Version) bool

// SelectVersion calls f(version).
func (f VersionFilterFunc) SelectVersion(version *Version) bool { return f(version) }

// A RepoFilter filters a set of repos to only those for which SelectRepo
// returns true.
type RepoFilter interface {
	SelectRepo(string) bool
}

type repoFilters []RepoFilter

func (fs repoFilters) SelectRepo(repo string) bool {
	for _, f := range fs {
		if !f.SelectRepo(repo) {
			return false
		}
	}
	return true
}

// A RepoFilterFunc is a RepoFilter that selects only those repos for
// which the func returns true.
type RepoFilterFunc func(string) bool

// SelectRepo calls f(repo).
func (f RepoFilterFunc) SelectRepo(repo string) bool { return f(repo) }

// ByUnitsFilter is implemented by filters that restrict their
// selections to items that are in a set of source units. It allows
// the store to optimize calls by skipping data that it knows is not
// any of the the specified source units.
type ByUnitsFilter interface {
	ByUnits() []unit.ID2
}

// ByUnits creates a new filter that matches objects in any of the
// given source units. It panics if any of the unit IDs' names or
// types are empty.
func ByUnits(units ...unit.ID2) interface {
	DefFilter
	RefFilter
	UnitFilter
	ByUnitsFilter
} {
	for _, u := range units {
		if u.Name == "" {
			panic("unit.Name: empty")
		}
		if u.Type == "" {
			panic("unit.Type: empty")
		}
		if strings.Contains(u.Type, "/") {
			log.Printf("WARNING: srclib store.ByUnits was called with a source unit type of %q, which resembles a unit *name*. Did you mix up the order of ByUnits's arguments?", u.Type)
		}
	}
	return byUnitFilter(units)
}

type byUnitFilter []unit.ID2

func (f byUnitFilter) contains(u unit.ID2) bool {
	for _, uu := range f {
		if uu == u {
			return true
		}
	}
	return false
}

func (f byUnitFilter) ByUnits() []unit.ID2 { return f }
func (f byUnitFilter) SelectDef(def *graph.Def) bool {
	return (def.Unit == "" && def.UnitType == "") || f.contains(unit.ID2{Type: def.UnitType, Name: def.Unit})
}
func (f byUnitFilter) SelectRef(ref *graph.Ref) bool {
	return (ref.Unit == "" && ref.UnitType == "") || f.contains(unit.ID2{Type: ref.UnitType, Name: ref.Unit})
}
func (f byUnitFilter) SelectUnit(unit *unit.SourceUnit) bool {
	return (unit.Type == "" && unit.Name == "") || f.contains(unit.ID2())
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
	return def.CommitID == "" || def.CommitID == f.commitID
}
func (f byCommitIDFilter) SelectRef(ref *graph.Ref) bool {
	return ref.CommitID == "" || ref.CommitID == f.commitID
}
func (f byCommitIDFilter) SelectUnit(unit *unit.SourceUnit) bool {
	return unit.CommitID == "" || unit.CommitID == f.commitID
}
func (f byCommitIDFilter) SelectVersion(version *Version) bool {
	return version.CommitID == "" || version.CommitID == f.commitID
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
	return def.Repo == "" || def.Repo == f.repo
}
func (f byRepoFilter) SelectRef(ref *graph.Ref) bool {
	return ref.Repo == "" || ref.Repo == f.repo
}
func (f byRepoFilter) SelectUnit(unit *unit.SourceUnit) bool {
	return unit.Repo == "" || unit.Repo == f.repo
}
func (f byRepoFilter) SelectVersion(version *Version) bool {
	return version.Repo == "" || version.Repo == f.repo
}
func (f byRepoFilter) SelectRepo(repo string) bool {
	return repo == f.repo
}

// ByRepoAndCommitID returns a filter by both repo and commit ID (both
// must match for an item to be selected by this filter). It panics if
// either repo or commitID is empty.
func ByRepoAndCommitID(repo, commitID string) interface {
	DefFilter
	RefFilter
	UnitFilter
	VersionFilter
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
	return (version.Repo == "" || version.Repo == f.repo) && (version.CommitID == "" || version.CommitID == f.commitID)
}

// ByUnitKey returns a filter by a source unit key. It panics if any
// fields on the unit key are not set. To filter by only source unit
// name and type, use ByUnits.
func ByUnitKey(key unit.Key) interface {
	DefFilter
	RefFilter
	UnitFilter
	ByRepoFilter
	ByCommitIDFilter
	ByUnitsFilter
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

func (f byUnitKeyFilter) ByRepo() string      { return f.key.Repo }
func (f byUnitKeyFilter) ByCommitID() string  { return f.key.CommitID }
func (f byUnitKeyFilter) ByUnits() []unit.ID2 { return []unit.ID2{f.key.ID2()} }
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
	ByUnitsFilter
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
func (f byDefKeyFilter) ByUnits() []unit.ID2 {
	return []unit.ID2{{Type: f.key.UnitType, Name: f.key.Unit}}
}
func (f byDefKeyFilter) ByDefPath() string { return f.key.Path }
func (f byDefKeyFilter) SelectDef(def *graph.Def) bool {
	return (def.Repo == "" || def.Repo == f.key.Repo) && (def.CommitID == "" || def.CommitID == f.key.CommitID) &&
		(def.UnitType == "" || def.UnitType == f.key.UnitType) && (def.Unit == "" || def.Unit == f.key.Unit) &&
		def.Path == f.key.Path
}

// ByRefDefFilter is implemented by filters that restrict their
// selection to refs with a specific target definition.
type ByRefDefFilter interface {
	ByDefRepo() string
	ByDefUnitType() string
	ByDefUnit() string
	ByDefPath() string
}

// ByRefDef returns a filter by ref target def. It panics if any
// fields on the target def are not set.
func ByRefDef(def graph.RefDefKey) RefFilter {
	if def.DefRepo == "" {
		panic("def.DefRepo: empty")
	}
	if def.DefUnitType == "" {
		panic("def.DefUnitType: empty")
	}
	if def.DefUnit == "" {
		panic("def.DefUnit: empty")
	}
	if def.DefPath == "" {
		panic("def.DefPath: empty")
	}
	return &byRefDefFilter{def: def}
}

type byRefDefFilter struct {
	def graph.RefDefKey

	// These fields hold the repo and source unit that the filter is
	// being applied to. This workaround is necessary because
	// otherwise the filter would have no way to match the Ref.DefXyz
	// fields, since they are zeroed out if they equal the values for
	// the current repo and source unit.
	//
	// Unlike for other filters, a ByRefDef filter can't be used
	// (currently) to narrow the scope since a ref's def does not
	// determine where the ref is stored.

	impliedRepo string   // the implied DefRepo value when ref.DefRepo == ""
	impliedUnit unit.ID2 // the implied DefUnit{,Type} value when ref.DefUnit{,Type} == ""
}

func (f *byRefDefFilter) ByDefRepo() string          { return f.def.DefRepo }
func (f *byRefDefFilter) ByDefUnitType() string      { return f.def.DefUnitType }
func (f *byRefDefFilter) ByDefUnit() string          { return f.def.DefUnit }
func (f *byRefDefFilter) ByDefPath() string          { return f.def.DefPath }
func (f *byRefDefFilter) setImpliedRepo(repo string) { f.impliedRepo = repo }
func (f *byRefDefFilter) setImpliedUnit(u unit.ID2)  { f.impliedUnit = u }
func (f *byRefDefFilter) SelectRef(ref *graph.Ref) bool {
	return ((ref.DefRepo == "" && f.impliedRepo == f.def.DefRepo) || ref.DefRepo == f.def.DefRepo) &&
		((ref.DefUnitType == "" && f.impliedUnit.Type == f.def.DefUnitType) || ref.DefUnitType == f.def.DefUnitType) &&
		((ref.DefUnit == "" && f.impliedUnit.Name == f.def.DefUnit) || ref.DefUnit == f.def.DefUnit) &&
		ref.DefPath == f.def.DefPath
}

var _ impliedRepoSetter = (*byRefDefFilter)(nil)
var _ impliedUnitSetter = (*byRefDefFilter)(nil)

// An AbsRefFilterFunc creates a RefFilter that selects only those
// refs for which the func returns true. Unlike RefFilterFunc, the
// ref's Def{Repo,UnitType,Unit,Path}, Repo, and CommitID fields are
// populated.
//
// AbsRefFilterFunc is less efficient than RefFilterFunc because it
// must make a copy of each ref before passing it to the func. If you
// don't need to access any of the fields it sets, use a
// RefFilterFunc.
func AbsRefFilterFunc(f RefFilterFunc) RefFilter {
	if f == nil {
		panic("AbsRefFilterFunc: f == nil")
	}
	return &absRefFilterFunc{f: f}
}

type absRefFilterFunc struct {
	f RefFilterFunc

	impliedRepo     string   // the implied DefRepo/Repo value when those are empty
	impliedCommitID string   // the CommitID currently being filtered
	impliedUnit     unit.ID2 // the implied DefUnitType/UnitType value when those are empty
}

func (f *absRefFilterFunc) setImpliedRepo(repo string)         { f.impliedRepo = repo }
func (f *absRefFilterFunc) setImpliedCommitID(commitID string) { f.impliedCommitID = commitID }
func (f *absRefFilterFunc) setImpliedUnit(u unit.ID2)          { f.impliedUnit = u }
func (f *absRefFilterFunc) SelectRef(ref *graph.Ref) bool {
	copy := *ref
	copy.Repo = f.impliedRepo
	copy.UnitType = f.impliedUnit.Type
	copy.Unit = f.impliedUnit.Name
	copy.CommitID = f.impliedCommitID
	if copy.DefRepo == "" {
		copy.DefRepo = f.impliedRepo
	}
	if copy.DefUnitType == "" {
		copy.DefUnitType = f.impliedUnit.Type
	}
	if copy.DefUnit == "" {
		copy.DefUnit = f.impliedUnit.Name
	}
	return f.f(&copy)
}

var _ impliedRepoSetter = (*absRefFilterFunc)(nil)
var _ impliedCommitIDSetter = (*absRefFilterFunc)(nil)
var _ impliedUnitSetter = (*absRefFilterFunc)(nil)

type impliedRepoSetter interface {
	setImpliedRepo(string)
}
type impliedCommitIDSetter interface {
	setImpliedCommitID(string)
}
type impliedUnitSetter interface {
	setImpliedUnit(unit.ID2)
}

func setImpliedRepo(fs []RefFilter, repo string) {
	for _, f := range fs {
		if f, ok := f.(impliedRepoSetter); ok {
			f.setImpliedRepo(repo)
		}
	}
}

func setImpliedCommitID(fs []RefFilter, commitID string) {
	for _, f := range fs {
		if f, ok := f.(impliedCommitIDSetter); ok {
			f.setImpliedCommitID(commitID)
		}
	}
}

func setImpliedUnit(fs []RefFilter, u unit.ID2) {
	for _, f := range fs {
		if f, ok := f.(impliedUnitSetter); ok {
			f.setImpliedUnit(u)
		}
	}
}

// ByDefPathFilter is implemented by filters that restrict their
// selection to defs with a specific def path.
type ByDefPathFilter interface {
	ByDefPath() string
}

// ByDefPath returns a filter by def path. It panics if defPath is
// empty.
func ByDefPath(defPath string) interface {
	DefFilter
	ByDefPathFilter
} {
	if defPath == "" {
		panic("defPath: empty")
	}
	return byDefPathFilter(defPath)
}

type byDefPathFilter string

func (f byDefPathFilter) ByDefPath() string { return string(f) }
func (f byDefPathFilter) SelectDef(def *graph.Def) bool {
	return def.Path == string(f)
}

func storeFilters(anyFilters interface{}) []interface{} {
	switch o := anyFilters.(type) {
	case DefFilter:
		return []interface{}{o}
	case []DefFilter:
		fs := make([]interface{}, len(o))
		for i, f := range o {
			fs[i] = f
		}
		return fs
	}

	v := reflect.ValueOf(anyFilters)
	if !v.IsValid() {
		// no filters
		return nil
	}

	switch v.Kind() {
	case reflect.Slice:
		if v.Len() == 0 {
			return nil
		}
		filters := make([]interface{}, v.Len())
		for i := 0; i < v.Len(); i++ {
			filters[i] = v.Index(i).Interface()
		}
		return filters

	default:
		return []interface{}{anyFilters}
	}
}
