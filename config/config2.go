package config

import (
	"sourcegraph.com/sourcegraph/srclib"
	"sourcegraph.com/sourcegraph/srclib/graph2"
)

type Tree2 struct {
	Units []*graph2.Unit `json:",omitempty"`

	// Scanners to use to scan for source units in this tree.
	Scanners []*srclib.ToolRef `json:",omitempty"`

	///// TODO:

	// SkipDirs is a list of directory trees that are skipped. That is, any
	// source units (produced by scanners) whose Dir is in a skipped dir tree is
	// not processed further.
	SkipDirs []string `json:",omitempty"`

	// SkipUnits is a list of source units that are skipped. That is,
	// any scanned source units whose name and type exactly matches a
	// name and type pair in SkipUnits is skipped.
	SkipUnits []struct{ Name, Type string } `json:",omitempty"`

	// TODO(sqs): Add some type of field that lets the Srcfile and the scanners
	// have input into which tools get used during the execution phase. Right
	// now, we're going to try just using the system defaults (srclib-*) and
	// then add more flexibility when we are more familiar with the system.

	// Config is an arbitrary key-value property map. Properties are copied
	// verbatim to each source unit that is scanned in this tree.
	Config map[string]interface{} `json:",omitempty"`
}
