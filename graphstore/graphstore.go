package graphstore

import (
	"encoding/json"
	"os"
	"path/filepath"

	"sort"

	"sourcegraph.com/sourcegraph/rwvfs"
	"sourcegraph.com/sourcegraph/srclib/graph"
)

// StoreDirName is the name of the directory in which all repository
// build data is stored, relative to the user's .srclib directory. We
// will eventually remove the .srclib-cache directory and use the
// graph store exclusively.
var StoreDirName = "store"

// TODO(graphstore):
// * write doc.go
// * store defs
// * store docs

// Graph store layout
// ------------------
// <defs> := SRCLIBPATH/defs/<def-path>
// <def-path> := <repo>/<unit-type>/<unit>/<path>/<commit-id>
// <def-path-no-commit-id> := <repo>/<unit-type>/<unit>/<path>
//
// <refs> := SRCLIBPATH/refs/<ref-path>
// <ref-path> := <def-path-no-commit-id>/.refs/<ref-repo>

// SAMER: docstring
type Store struct {
	fs rwvfs.WalkableFileSystem
}

// New takes a path for the global srclib directory and returns a Store.
func New(srclibPath string) (*Store, error) {
	storeDir := filepath.Join(srclibPath, StoreDirName)
	if err := os.Mkdir(storeDir, 0700); err != nil && !os.IsExist(err) {
		return nil, err
	}
	return &Store{rwvfs.Walkable(rwvfs.OS(storeDir))}, nil
}

var graphStore Store

// constructDefPath constructs a file path for DefKey d. When includeCommitID
// is false, the commit id is not used in the path. constructDefPath
// returns the empty string if d is incorrectly formed.
//
// A def path can have one of the following forms:
//     <def-path> := <repo>/<unit-type>/<unit>/<path>/<commit-id>
//     <def-path-no-commit-id> := <repo>/<unit-type>/<unit>/<path>
func (s *Store) constructDefPath(d graph.DefKey, includeCommitID bool) string {
	if d.Repo == "" || d.UnitType == "" || d.Unit == "" || d.Path == "" {
		return ""
	}
	p := s.fs.Join(d.Repo, d.UnitType, d.Unit, string(d.Path))
	if includeCommitID {
		if d.CommitID == "" {
			return ""
		}
		p = s.fs.Join(p, d.CommitID)
	}
	return p
}

func (s *Store) refsFS(d graph.DefKey) rwvfs.WalkableFileSystem {
	p := s.constructDefPath(d, false)
	return rwvfs.Walkable(rwvfs.Sub(s.fs, s.fs.Join("refs", p, ".refs")))
}

// SAMER: this stuff should go in refs.go... or defs.go
type ListRefsOptions struct {
	// Repo is a repository URI. When it is not blank, only fetch
	// references from Repo.
	Repo string `url:",omitempty"`
}

const RefsGlob = "*.refs"

// ListRefs lists the references for the definition specified by d.
// ListRefs is commit agnostic, and d's commit id is ignored.
func (s *Store) ListRefs(d graph.DefKey, opt *ListRefsOptions) ([]*graph.Ref, error) {
	if opt == nil {
		opt = &ListRefsOptions{}
	}
	f := s.refsFS(d)
	matches, err := rwvfs.Glob(f, opt.Repo, RefsGlob)
	if err != nil {
		return nil, err
	}
	// TODO(samer): preallocate space?
	var refs []*graph.Ref
	// Read in all refs.
	for _, m := range matches {
		// TODO(samer): move Join out of loop.
		file, err := f.Open(filepath.Join(opt.Repo, m))
		if err != nil {
			return nil, err
		}
		rs := &[]*graph.Ref{}
		if err := json.NewDecoder(file).Decode(rs); err != nil {
			return nil, err
		}
		refs = append(refs, *rs...)
	}
	return refs, nil
}

type sortableRefs struct{ refs []*graph.Ref }

func (rs sortableRefs) Len() int {
	return len(rs.refs)
}

func (rs sortableRefs) Less(i, j int) bool {
	return rs.refs[i].RefDefKey().DefRepo < rs.refs[j].RefDefKey().DefRepo
}

func (rs sortableRefs) Swap(i, j int) {
	rs.refs[i], rs.refs[j] = rs.refs[j], rs.refs[i]
}

// StoreRefs stores the refs in the graph store.
func (s *Store) StoreRefs(refs []*graph.Ref) error {
	sortable := sortableRefs{refs}
	sort.Sort(sortable)
	var prevRepo string
	var prevRefs []*graph.Ref
	for i, r := range sortable.refs {
		if i == len(sortable.refs)-1 || r.RefDefKey().DefRepo != prevRepo && len(prevRefs) > 0 {
			f := s.refsFS(r.DefKey())
			refsFile, err := f.Create(r.RefDefKey().DefRepo + "/all.refs")
			if err != nil {
				return err
			}
			if err := json.NewEncoder(refsFile).Encode(prevRefs); err != nil {
				return err
			}
			prevRepo = r.RefDefKey().DefRepo
			prevRefs = []*graph.Ref{r}
			continue
		}
		prevRefs = append(prevRefs, r)
	}
	return nil
}
