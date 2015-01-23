package store

import "testing"

type mockDefIndex struct {
	Covers_ func([]DefFilter) int
	Defs_   func(...DefFilter) (byteOffsets, error)
	name    string
}

func (m mockDefIndex) Covers(fs []DefFilter) int                 { return m.Covers_(fs) }
func (m mockDefIndex) Defs(fs ...DefFilter) (byteOffsets, error) { return m.Defs_(fs...) }

func TestBestCoverageDefIndex(t *testing.T) {
	tests := map[string]struct {
		indexes           []interface{}
		wantBestIndexName string
	}{
		"empty indexes": {
			indexes:           []interface{}{},
			wantBestIndexName: "",
		},
		"coverage 0": {
			indexes:           []interface{}{mockDefIndex{Covers_: func([]DefFilter) int { return 0 }}},
			wantBestIndexName: "",
		},
		"coverage 1": {
			indexes: []interface{}{
				mockDefIndex{
					Covers_: func([]DefFilter) int { return 1 },
					name:    "1",
				},
			},
			wantBestIndexName: "1",
		},
		"choose index with highest coverage": {
			indexes: []interface{}{
				mockDefIndex{
					Covers_: func([]DefFilter) int { return 2 },
					name:    "2",
				},
				mockDefIndex{
					Covers_: func([]DefFilter) int { return 1 },
					name:    "1",
				},
			},
			wantBestIndexName: "2",
		},
	}
	for label, test := range tests {
		// Filters don't matter for this test since we just call
		// (defIndex).Covers.
		dx := bestCoverageDefIndex(test.indexes, nil)
		var name string
		if dx != nil {
			name = dx.(mockDefIndex).name
		}
		if name != test.wantBestIndexName {
			t.Errorf("%s: got best index %q, want %q", label, name, test.wantBestIndexName)
		}
	}
}
