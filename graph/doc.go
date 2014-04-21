package graph

import (
	"sourcegraph.com/sourcegraph/db"
	"sourcegraph.com/sourcegraph/repo"
)

// Docstring
type Doc struct {
	SymbolKey

	Format string
	Data   string

	File  string `json:"file"`
	Start int    `json:"start"`
	End   int    `json:"end"`
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
	Title string `json:"title"` // Doc title
	Body  string `json:"body"`  // HTML tags with the data-sg-doc-symbol attribute will be linked to symbol pages and vice-versa in the UI
	Toc   string `json:"toc"`   // Table of contents in conjunction (in sidebar) with body

	SymbolPaths *db.StringSlice `json:"symbolPaths"` // symbols within the scope of this documentation page
}
