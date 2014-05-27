package graph

import (
	"sourcegraph.com/sourcegraph/srcgraph/db_common"
	"sourcegraph.com/sourcegraph/srcgraph/repo"
)

// Docstring
type Doc struct {
	SymbolKey

	Format string
	Data   string

	File  string
	Start int
	End   int
}

func (d *Doc) sortKey() string { return d.SymbolKey.String() }

// Sorting

type Docs []*Doc

func (vs Docs) Len() int           { return len(vs) }
func (vs Docs) Swap(i, j int)      { vs[i], vs[j] = vs[j], vs[i] }
func (vs Docs) Less(i, j int) bool { return vs[i].sortKey() < vs[j].sortKey() }

type DocPageKey struct {
	Repo     repo.URI
	UnitType string `db:"unit_type"`
	Unit     string
	Path     string
}

type DocPage struct {
	DocPageKey

	// Note: the contents of these fields is unsanitized. Any sanitization should be done in the UI.
	Title string // Doc title
	Body  string // HTML tags with the data-sg-doc-symbol attribute will be linked to symbol pages and vice-versa in the UI
	Toc   string // Table of contents in conjunction (in sidebar) with body

	SymbolPaths *db_common.StringSlice // symbols within the scope of this documentation page
}
