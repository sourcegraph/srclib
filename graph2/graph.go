package graph2

import "sourcegraph.com/sourcegraph/srclib/buildstore"

func init() {
	buildstore.RegisterDataType("unit2", Unit{})
}
