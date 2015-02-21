package graph

import "encoding/json"

// Key returns the unique key for the doc.
func (d *Doc) Key() DocKey {
	return DocKey{DefKey: d.DefKey, Format: d.Format, Start: d.Start}
}

// DocKey is the unique key for a doc. Each doc within a source unit
// must have a unique DocKey.
//
// Freestanding comments will not have an associated DefKey, but they
// *must* provide 'Start' and 'End', where 'Start' != 'End'.
type DocKey struct {
	DefKey
	Format string
	Start  uint32
	End    uint32
}

func (d DocKey) String() string {
	b, _ := json.Marshal(d)
	return string(b)
}

func (d *Doc) sortKey() string { return d.Key().String() }

// Sorting

type Docs []*Doc

func (vs Docs) Len() int           { return len(vs) }
func (vs Docs) Swap(i, j int)      { vs[i], vs[j] = vs[j], vs[i] }
func (vs Docs) Less(i, j int) bool { return vs[i].sortKey() < vs[j].sortKey() }
