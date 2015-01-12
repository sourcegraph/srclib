package graphstore

import (
	"encoding/json"
	"path/filepath"

	"sort"

	"sourcegraph.com/sourcegraph/rwvfs"
	"sourcegraph.com/sourcegraph/srclib/graph"
)

// StoreDirName is the name of the directory in which all repository
// build data is stored, relative to the user's .srclib directory. We
// will eventually remove the .srclib-cache directory and use the
// store exclusively.
var StoreDirName = "store"

// what I need:
// * walk srclib-cache
// * write doc.go

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
	FS rwvfs.WalkableFileSystem
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
	p := s.Join(d.Repo, d.UnitType, d.Unit, d.Path)
	if includeCommitID {
		if d.CommitID == "" {
			return ""
		}
		p = s.Join(p, d.CommitID)
	}
	return p
}

func (s *Store) refsFS(d graph.DefKey) rwvfs.WalkableFileSystem {
	p := s.constructDefPath(d, false)
	return fs.Walkable(rwvfs.Sub(s, s.Join("refs", p, ".refs")))
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
		file, err := f.Open(m)
		if err != nil {
			return nil, err
		}
		rs := &[]*graph.Ref{}
		// TODO(samer): move Join out of loop.
		d := json.Decoder(filepath.Join(opt.Repo, file))
		if err := d.Decode(rs); err != nil {
			return nil, err
		}
		refs = append(refs, rs)
	}
	return refs
}

var sortableRefs struct{ refs []*graph.Ref }

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
	sortable = sortableRefs{refs}
	sort.Sort(sortable)
	var prevRepo string
	var prevRefs []*graph.Ref
	for i, r := range sortable.refs {
		if i == len(sortable.refs)-1 || r.RefDefKey().Repo != prevRepo && len(prevRefs) > 0 {
			f := s.refsFS(r.DefKey())
			refsFile, err := f.Open(r.RefDefKey().Repo + "/all.refs")
			if err != nil {
				return err
			}
			if err := json.NewEncoder(refsFile).Encode(prevRefs); err != nil {
				return err
			}
			prevRepo = r.RefDefKey().Repo
			prevRefs = []*graph.Ref{r}
			continue
		}
		prevRefs = append(prevRefs, r)
	}
}
