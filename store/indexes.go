package store

import (
	"log"
	"os"
	"reflect"
	"time"

	"code.google.com/p/rog-go/parallel"

	"strings"
	"sync"

	"sort"

	"sourcegraph.com/sourcegraph/srclib/unit"
)

// IndexStatus describes an index and its status (whether it exists,
// etc.).
type IndexStatus struct {
	// Repo is the ID of the repository this index pertains to. If it
	// pertains to all repositories in a MultiRepoStore, or if it
	// pertains to the current (and only) repository in a RepoStore or
	// lower-level store, the Repo field is empty.
	Repo string `json:",omitempty"`

	// CommitID is the commit ID of the version this index pertains to. If it
	// pertains to all commits in a RepoStore, or if it
	// pertains to the current (and only) commit in a TreeStore or
	// lower-level store, the CommitID field is empty.
	CommitID string `json:",omitempty"`

	// Unit is the commit ID of the version this index pertains to. If
	// it pertains to all units in a TreeStore, or if it pertains to
	// the current (and only) source unit in a UnitStore, the Unit
	// field is empty.
	Unit *unit.ID2 `json:",omitempty"`

	// Stale is a boolean value indicating whether the index needs to
	// be (re)built.
	Stale bool

	// Name is the name of the index.
	Name string

	// Type is the type of the index.
	Type string

	// Size is the length in bytes of the index if it is a regular
	// file.
	Size int64 `json:",omitempty"`

	// Error is the error encountered while determining this index's
	// status, if any.
	Error string `json:",omitempty"`

	// DependsOnChildren is true if this index needs its child indexes
	// to be built first before it can be built.
	DependsOnChildren bool `json:",omitempty"`

	// BuildError is the error encountered while building this index,
	// if any. It is only returned by BuildIndexes (not Indexes).
	BuildError string `json:",omitempty"`

	// BuildDuration is how long it took to build the index. It is
	// only returned by BuildIndexes (not Indexes).
	BuildDuration time.Duration `json:",omitempty"`
}

// IndexCriteria restricts a set of indexes to only those that match
// the criteria. Non-empty conditions are ANDed together.
type IndexCriteria struct {
	Repo     string
	CommitID string
	Unit     *unit.ID2
	Name     string
	Type     string
	Stale    *bool

	ReposLimit  int
	ReposOffset int
}

// BuildIndexes builds all indexes on store and its lower-level stores
// that match the specified criteria. It returns the status of each
// index that was built (or rebuilt).
func BuildIndexes(store interface{}, c IndexCriteria, indexChan chan<- IndexStatus) ([]IndexStatus, error) {
	var built []IndexStatus
	var builtMu sync.Mutex
	indexChan2 := make(chan storeIndex)
	done := make(chan struct{})
	go func() {
		var par *parallel.Run
		lastDependsOnChildren := false
		for sx := range indexChan2 {
			doBuild := func(sx storeIndex) {
				start := time.Now()
				err := sx.store.BuildIndex(sx.status.Name, sx.index)
				sx.status.BuildDuration = time.Since(start)
				if err == nil {
					sx.status.Stale = false
				} else {
					sx.status.BuildError = err.Error()
				}
				builtMu.Lock()
				built = append(built, sx.status)
				builtMu.Unlock()
				if indexChan != nil {
					indexChan <- sx.status
				}
			}

			// Run indexes in parallel, but if we
			// encounter an index that depends on children, wait for
			// all previously seen indexes to finish before building
			// those indexes.
			if sx.status.DependsOnChildren != lastDependsOnChildren && par != nil {
				par.Wait()
				par = nil
			}
			if par == nil {
				par = parallel.NewRun(MaxIndexParallel)
			}
			sx_ := sx
			par.Do(func() error { doBuild(sx_); return nil })

			lastDependsOnChildren = sx.status.DependsOnChildren
		}
		if par != nil {
			par.Wait()
		}
		done <- struct{}{}
	}()
	err := listIndexes(store, c, indexChan2, nil)
	close(indexChan2)
	<-done
	return built, err
}

// Indexes returns a list of indexes and their statuses for store and
// its lower-level stores. Only indexes matching the criteria are
// returned. If indexChan is non-nil, it receives indexes as soon as
// they are found; when all matching indexes have been found, the func
// returns and all indexes are included in the returned slice.
//
// The caller is responsible for closing indexChan after Indexes
// returns (if desired).
func Indexes(store interface{}, c IndexCriteria, indexChan chan<- IndexStatus) ([]IndexStatus, error) {
	var xs []IndexStatus
	indexChan2 := make(chan storeIndex)
	done := make(chan struct{})
	go func() {
		for sx := range indexChan2 {
			xs = append(xs, sx.status)
			if indexChan != nil {
				indexChan <- sx.status
			}
		}
		done <- struct{}{}
	}()
	err := listIndexes(store, c, indexChan2, nil)
	close(indexChan2)
	<-done
	return xs, err
}

// storeIndex is a (store,index) pair that makes it easy to perform
// both both index status listing and index creation. It is sent on
// the channel by listIndexes to its callers.
type storeIndex struct {
	store  indexedStore
	status IndexStatus
	index  Index
}

// listIndexes lists indexes in s (a store) asynchronously, sending
// status objects to ch. If f != nil, it is called to set/modify
// fields on each status object before the storeIndex object is sent to
// the channel.
func listIndexes(s interface{}, c IndexCriteria, ch chan<- storeIndex, f func(*IndexStatus)) error {
	switch s := s.(type) {
	case indexedStore:
		xx := s.Indexes()
		var waitingOnChildren []storeIndex
		for name, x := range xx {
			st := IndexStatus{
				Name: name,
				Type: strings.TrimPrefix(reflect.TypeOf(x).String(), "*store."),
			}

			if !strings.Contains(st.Name, c.Name) {
				continue
			}
			if !strings.Contains(st.Type, c.Type) {
				continue
			}

			fi, err := s.statIndex(name)
			if os.IsNotExist(err) {
				st.Stale = true
			} else if err != nil {
				st.Error = err.Error()
			} else {
				st.Size = fi.Size()
			}

			switch x.(type) {
			case unitRefIndexBuilder, defQueryTreeIndexBuilder:
				st.DependsOnChildren = true
			}

			if c.Stale != nil && st.Stale != *c.Stale {
				continue
			}

			si := storeIndex{store: s, status: st, index: x}
			if f != nil {
				f(&si.status)
			}
			if st.DependsOnChildren {
				waitingOnChildren = append(waitingOnChildren, si)
			} else {
				ch <- si
			}
		}

		switch s := s.(type) {
		case *indexedTreeStore:
			if err := listIndexes(s.fsTreeStore, c, ch, f); err != nil {
				return err
			}
		case *indexedUnitStore:
			if err := listIndexes(s.fsUnitStore, c, ch, f); err != nil {
				return err
			}
		}

		for _, si := range waitingOnChildren {
			ch <- si
		}

	case repoStoreOpener:
		var rss map[string]RepoStore
		if c.Repo == "" {
			var err error
			rss, err = s.openAllRepoStores()
			if err != nil && !isStoreNotExist(err) {
				return err
			}
		} else {
			rss = map[string]RepoStore{c.Repo: s.openRepoStore(c.Repo)}
		}

		// Sort repos for determinism.
		repos := make([]string, 0, len(rss))
		for repo := range rss {
			repos = append(repos, repo)
		}
		sort.Strings(repos)

		if c.ReposOffset != 0 {
			if c.ReposOffset < len(repos) {
				repos = repos[c.ReposOffset:]
			} else {
				log.Printf("Warning: A ReposOffset (%d) was specified that equals or exceeds the total number of repos (%d).", c.ReposOffset, len(repos))
			}
		}
		if c.ReposLimit != 0 && c.ReposLimit < len(repos) {
			repos = repos[:c.ReposLimit]
		}

		for _, repo := range repos {
			rs := rss[repo]
			err := listIndexes(rs, c, ch, func(x *IndexStatus) {
				x.Repo = repo
				if f != nil {
					f(x)
				}
			})
			if err != nil {
				return err
			}
		}

	case treeStoreOpener:
		var tss map[string]TreeStore
		if c.CommitID == "" {
			var err error
			tss, err = s.openAllTreeStores()
			if err != nil && !isStoreNotExist(err) {
				return err
			}
		} else {
			tss = map[string]TreeStore{c.CommitID: s.openTreeStore(c.CommitID)}
		}
		for commitID, ts := range tss {
			err := listIndexes(ts, c, ch, func(x *IndexStatus) {
				x.CommitID = commitID
				if f != nil {
					f(x)
				}
			})
			if err != nil {
				return err
			}
		}

	case unitStoreOpener:
		var uss map[unit.ID2]UnitStore
		if c.Unit == nil {
			var err error
			uss, err = s.openAllUnitStores()
			if err != nil && !isStoreNotExist(err) {
				return err
			}
		} else {
			uss = map[unit.ID2]UnitStore{*c.Unit: s.openUnitStore(*c.Unit)}
		}
		if len(uss) > 0 {
			par := parallel.NewRun(MaxIndexParallel)
			for unit_, us_ := range uss {
				unit, us := unit_, us_
				par.Do(func() error {
					unitCopy := unit
					return listIndexes(us, c, ch, func(x *IndexStatus) {
						x.Unit = &unitCopy
						if f != nil {
							f(x)
						}
					})
				})
			}
			if err := par.Wait(); err != nil {
				return err
			}
		}

	}
	return nil
}

var MaxIndexParallel = 1
