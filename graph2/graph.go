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

// Sorting

type Nodes []*Node

func (l Nodes) Len() int           { return len(l) }
func (l Nodes) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }
func (l Nodes) Less(i, j int) bool { return l[i].NodeKey.String() < l[j].NodeKey.String() }

type Edges []*Edge

func (l Edges) Len() int           { return len(l) }
func (l Edges) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }
func (l Edges) Less(i, j int) bool { return l[i].String() < l[j].String() }
