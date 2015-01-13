package graphstore

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/kr/fs"

	"sort"
	"strings"

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

// Store represents the user's graph store.
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
	p := s.fs.Join("refs", s.constructDefPath(d, false), ".refs")
	if err := rwvfs.MkdirAll(s.fs, p); err != nil {
		panic(err)
	}
	return rwvfs.Walkable(rwvfs.Sub(s.fs, p))
}

type ListRefsOptions struct {
	// Repo is a repository URI. When it is not blank, only fetch
	// references from Repo.
	Repo string `url:",omitempty"`
}

const RefsSuffix = ".refs"

// ListRefs lists the references for the definition specified by d.
// ListRefs is commit agnostic, and d's commit id is ignored.
func (s *Store) ListRefs(d graph.DefKey, opt *ListRefsOptions) ([]*graph.Ref, error) {
	if opt == nil {
		opt = &ListRefsOptions{}
	}
	f := s.refsFS(d)

	var refFiles []string
	walker := fs.WalkFS(opt.Repo, f)
	for walker.Step() {
		if strings.HasSuffix(walker.Path(), RefsSuffix) {
			refFiles = append(refFiles, walker.Path())
		}
	}
	// TODO(samer): preallocate space?
	var refs []*graph.Ref
	// Read in all refs.
	for _, rf := range refFiles {
		file, err := f.Open(rf)
		if err != nil {
			return nil, err
		}
		rs := &[]*graph.Ref{}
		if err := json.NewDecoder(file).Decode(rs); err != nil {
			return nil, err
		}
		if err := file.Close(); err != nil {
			return nil, err
		}
		refs = append(refs, *rs...)
	}
	return refs, nil
}

type refsSortableByRefDefKey struct{ refs []*graph.Ref }

func (rs refsSortableByRefDefKey) Len() int      { return len(rs.refs) }
func (rs refsSortableByRefDefKey) Swap(i, j int) { rs.refs[i], rs.refs[j] = rs.refs[j], rs.refs[i] }
func (rs refsSortableByRefDefKey) Less(i, j int) bool {
	return rs.refs[i].RefDefKey().String() < rs.refs[j].RefDefKey().String()
}

type refsSortableByRepo struct{ refs []*graph.Ref }

func (rs refsSortableByRepo) Len() int           { return len(rs.refs) }
func (rs refsSortableByRepo) Swap(i, j int)      { rs.refs[i], rs.refs[j] = rs.refs[j], rs.refs[i] }
func (rs refsSortableByRepo) Less(i, j int) bool { return rs.refs[i].Repo < rs.refs[j].Repo }

// StoreRefs stores the refs in the graph store.
func (s *Store) StoreRefs(refs []*graph.Ref) error {
	writeRefs := func(f rwvfs.WalkableFileSystem, refs []*graph.Ref) error {
		if len(refs) == 0 {
			return nil
		}
		err := rwvfs.MkdirAll(f, refs[0].Repo) // All members of refs have the same Repo.
		if err != nil {
			return err
		}
		refsFile, err := f.Create(refs[0].Repo + "/all.refs")
		if err != nil {
			return err
		}
		if err := json.NewEncoder(refsFile).Encode(refs); err != nil {
			return err
		}
		if err := refsFile.Close(); err != nil {
			return err
		}
		return nil
	}
	writeRefsToDefKey := func(refs []*graph.Ref) error {
		if len(refs) == 0 {
			return nil
		}
		f := s.refsFS(refs[0].DefKey()) // All members of refs have the same DefKey.
		sortable := refsSortableByRepo{refs}
		sort.Sort(sortable)
		var prevRepo string
		var prevRefs []*graph.Ref
		for _, ref := range sortable.refs {
			if ref.Repo != prevRepo {
				if err := writeRefs(f, prevRefs); err != nil {
					return err
				}
				prevRepo = ref.Repo
				prevRefs = []*graph.Ref{ref}
				continue
			}
			prevRefs = append(prevRefs, ref)
		}
		return writeRefs(f, prevRefs)
	}
	sortable := refsSortableByRefDefKey{refs}
	sort.Sort(sortable)
	var prevRefDefKey graph.RefDefKey
	var prevRefs []*graph.Ref
	for _, r := range sortable.refs {
		if r.RefDefKey() != prevRefDefKey {
			if err := writeRefsToDefKey(prevRefs); err != nil {
				return err
			}
			prevRefDefKey = r.RefDefKey()
			prevRefs = []*graph.Ref{r}
			continue
		}
		prevRefs = append(prevRefs, r)
	}
	return writeRefsToDefKey(prevRefs)
}
