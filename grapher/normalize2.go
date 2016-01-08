package grapher

import (
	"sort"

	"github.com/sqs/fileset"
	"sourcegraph.com/sourcegraph/srclib/ann"
	"sourcegraph.com/sourcegraph/srclib/graph"
	"sourcegraph.com/sourcegraph/srclib/graph2"
)

// NormalizeData2 sorts data and performs other postprocessing.
func NormalizeData2(unitType, dir string, o *graph2.Output) error {
	for _, refEdge := range o.RefEdges {
		if refEdge.Src.URI != "" {
			uri, err := graph.TryMakeURI(refEdge.Src.URI)
			if err != nil {
				return err
			}
			refEdge.Src.URI = uri
		}

		if refEdge.Dst.URI != "" {
			uri, err := graph.TryMakeURI(refEdge.Dst.URI)
			if err != nil {
				return err
			}
			refEdge.Dst.URI = uri
		}
	}

	if unitType != "GoPackage" && unitType != "Dockerfile" && unitType != "NugetPackage" {
		ensureOffsetsAreByteOffsets2(dir, o)
	}

	if err := ValidateNodes(o.DefNodes, o.RefNodes, o.DocNodes, o.OtherNodes); err != nil {
		return err
	}
	if err := ValidateEdges(o.RefEdges, o.DocEdges, o.OtherEdges); err != nil {
		return err
	}

	sort.Sort(graph2.Nodes(o.DefNodes))
	sort.Sort(graph2.Nodes(o.RefNodes))
	sort.Sort(graph2.Nodes(o.DocNodes))
	sort.Sort(graph2.Nodes(o.OtherNodes))
	sort.Sort(graph2.Edges(o.RefEdges))
	sort.Sort(graph2.Edges(o.DocEdges))
	sort.Sort(graph2.Edges(o.OtherEdges))
	sort.Sort(ann.Anns(o.Anns))

	return nil
}

func ensureOffsetsAreByteOffsets2(dir string, o *graph2.Output) {
	fset := fileset.NewFileSet()
	files := make(map[string]*fileset.File)

	for _, s := range o.DefNodes {
		fixOffsets(dir, s.File, fset, files, &s.Start, &s.End)
	}
	for _, r := range o.RefNodes {
		fixOffsets(dir, r.File, fset, files, &r.Start, &r.End)
	}
	for _, d := range o.DocNodes {
		fixOffsets(dir, d.File, fset, files, &d.Start, &d.End)
	}
}
