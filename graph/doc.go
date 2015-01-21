package graph

import "encoding/json"

// START Doc OMIT
// Docstring
type Doc struct {
	// A link to the definition that this docstring describes
	DefKey

	// The MIME-type that the documentation is stored in. Valid formats include 'text/html', 'text/plain', 'text/x-markdown', text/x-rst'
	Format string

	// The actual documentation text
	Data string

	// Location where the docstring was extracted from. Leave blank for undefined location
	File  string
	Start int
	End   int
}

// Key returns the unique key for the doc.
func (d *Doc) Key() DocKey {
	return DocKey{DefKey: d.DefKey, Format: d.Format}
}

// DocKey is the unique key for a doc. Each doc within a source unit
// must have a unique DocKey.
type DocKey struct {
	DefKey
	Format string
}

func (d DocKey) String() string {
	b, _ := json.Marshal(d)
	return string(b)
}

// END Doc OMIT

func (d *Doc) sortKey() string { return d.Key().String() }

// Sorting

type Docs []*Doc

func (vs Docs) Len() int           { return len(vs) }
func (vs Docs) Swap(i, j int)      { vs[i], vs[j] = vs[j], vs[i] }
func (vs Docs) Less(i, j int) bool { return vs[i].sortKey() < vs[j].sortKey() }
