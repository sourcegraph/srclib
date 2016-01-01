package graph2

import "sourcegraph.com/sourcegraph/srclib/buildstore"

func init() {
	buildstore.RegisterDataType("unit2", Unit{})
}

func NewNodeKey(genus, uri, version, uname, utyp, path string) NodeKey {
	return NodeKey{
		UnitKey: UnitKey{
			TreeKey: TreeKey{
				Genus: genus,
				URI:   uri,
			},
			Version:  version,
			UnitName: uname,
			UnitType: utyp,
		},
		Path: path,
	}
}
