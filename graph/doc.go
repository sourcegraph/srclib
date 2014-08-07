package graph

import (
	"sourcegraph.com/sourcegraph/srclib/db_common"
	"sourcegraph.com/sourcegraph/srclib/repo"
)

// START Doc OMIT
// Docstring
type Doc struct {
	// A link to the definition that this docstring describes
	SymbolKey

	// The MIME-type that the documentation is stored in. Valid formats include 'text/html', 'text/plain', 'text/x-markdown', text/x-rst'
	Format string

	// The actual documentation text
	Data   string

	// Location where the docstring was extracted from. Leave blank for undefined location
	File  string
	Start int
	End   int
}
// END Doc OMIT

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
