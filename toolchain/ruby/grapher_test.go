package ruby

import (
	"testing"

	"sourcegraph.com/sourcegraph/srcgraph/graph"
)

func TestRubyPathToTreePath(t *testing.T) {
	tests := []struct {
		rubyPath string
		treePath graph.TreePath
	}{
		{"URI/$classmethods/parse", "URI/parse"},
		{"C>_local_0>f", "C/f"},
	}
	for _, test := range tests {
		treePath := rubyPathToTreePath(test.rubyPath)
		if treePath != test.treePath {
			t.Errorf("%q: got tree path %q, want %q", test.rubyPath, treePath, test.treePath)
		}
	}
}
