package graph2

import (
	"fmt"

	"sourcegraph.com/sourcegraph/srclib/buildstore"
)

func init() {
	buildstore.RegisterDataType("unit2", Unit{})
}

func NewNodeKey(treetype, uri, version, uname, utyp, path string) NodeKey {
	return NodeKey{
		UnitKey: UnitKey{
			TreeKey: TreeKey{
				TreeType: treetype,
				URI:      uri,
			},
			Version:  version,
			UnitName: uname,
			UnitType: utyp,
		},
		Path: path,
	}
}

// ID returns the build unit's unique ID within the source tree.
func (u *Unit) ID() string { return fmt.Sprintf("{%s %s}", u.UnitType, u.UnitName) }
