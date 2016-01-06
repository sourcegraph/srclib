package grapher

import (
	"fmt"

	"sourcegraph.com/sourcegraph/srclib/graph2"
)

func ValidateNodes(nodes ...[]*graph2.Node) (errs MultiError) {
	nodeKeys := make(map[graph2.NodeKey]struct{})
	for _, nodeList := range nodes {
		for _, node := range nodeList {
			key := node.NodeKey
			if _, in := nodeKeys[key]; in {
				errs = append(errs, fmt.Errorf("duplicate node key: %+v", key))
			} else {
				nodeKeys[key] = struct{}{}
			}
		}
	}
	return
}

func ValidateEdges(edges ...[]*graph2.Edge) (errs MultiError) {
	edgeKeys := make(map[graph2.Edge]struct{})
	for _, edgeList := range edges {
		for _, edge := range edgeList {
			if _, in := edgeKeys[*edge]; in {
				errs = append(errs, fmt.Errorf("duplicate edge: %+v", edge))
			} else {
				edgeKeys[*edge] = struct{}{}
			}
		}
	}
	return
}
