package graph

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"regexp"

	"sourcegraph.com/sourcegraph/go-nnz/nnz"
	"sourcegraph.com/sourcegraph/srclib/util/sqltypes"
)

type (
	DefPath  string
	TreePath string
)

// START DefKey OMIT
// DefKey specifies a definition, either concretely or abstractly. A concrete
// definition key has a non-empty CommitID and refers to a definition defined in a
// specific commit. An abstract definition key has an empty CommitID and is
// considered to refer to definitions from any number of commits (so long as the
// Repo, UnitType, Unit, and Path match).
//
// You can think of CommitID as the time dimension. With an empty CommitID, you
// are referring to a definition that may or may not exist at various times. With a
// non-empty CommitID, you are referring to a specific definition of a definition at
// the time specified by the CommitID.
type DefKey struct {
	// Repo is the VCS repository that defines this definition. Its Elasticsearch mapping is defined
	// separately.
	Repo string `json:",omitempty"`

	// CommitID is the ID of the VCS commit that this definition was defined in. The
	// CommitID is always a full commit ID (40 hexadecimal characters for git
	// and hg), never a branch or tag name.
	CommitID string `db:"commit_id" json:",omitempty"`

	// UnitType is the type name of the source unit (obtained from unit.Type(u))
	// that this definition was defined in.
	UnitType string `db:"unit_type" json:",omitempty"`

	// Unit is the name of the source unit (obtained from u.Name()) that this
	// definition was defined in.
	Unit string `json:",omitempty"`

	// Path is a unique identifier for the def, relative to the source unit.
	// It should remain stable across commits as long as the def is the
	// "same" def. Its Elasticsearch mapping is defined separately (because
	// it is a multi_field, which the struct tag can't currently represent).
	//
	// Path encodes no structural semantics. Its only meaning is to be a stable
	// unique identifier within a given source unit. In many languages, it is
	// convenient to use the namespace hierarchy (with some modifications) as
	// the Path, but this may not always be the case. I.e., don't rely on Path
	// to find parents or children or any other structural propreties of the
	// def hierarchy). See Def.TreePath instead.
	Path DefPath
}

// END DefKey OMIT

func (s DefKey) String() string {
	b, err := json.Marshal(s)
	if err != nil {
		panic("DefKey.String: " + err.Error())
	}
	return string(b)
}

// START Def OMIT
type Def struct {
	// DefKey is the natural unique key for a def. It is stable
	// (subsequent runs of a grapher will emit the same defs with the same
	// DefKeys).
	DefKey

	// TreePath is a structurally significant path descriptor for a def. For
	// many languages, it may be identical or similar to DefKey.Path.
	// However, it has the following constraints, which allow it to define a
	// def tree.
	//
	// A tree-path is a chain of '/'-delimited components. A component is either a
	// def name or a ghost component.
	// - A def name satifies the regex [^/-][^/]*
	// - A ghost component satisfies the regex -[^/]*
	// Any prefix of a tree-path that terminates in a def name must be a valid
	// tree-path for some def.
	// The following regex captures the children of a tree-path X: X(/-[^/]*)*(/[^/-][^/]*)
	TreePath TreePath `db:"treepath" json:",omitempty"`

	// Name of the definition. This need not be unique.
	Name string

	// Kind is the kind of thing this definition is. This is
	// language-specific. Possible values include "type", "func",
	// "var", etc.
	Kind string

	File string

	DefStart int `db:"def_start"`
	DefEnd   int `db:"def_end"`

	// Exported is whether this def is part of a source unit's
	// public API. For example, in Java a "public" field is
	// Exported.
	Exported bool

	// Local is whether this def is local to a function or some
	// other inner scope. Local defs do *not* have module,
	// package, or file scope. For example, in Java a function's
	// args are Local, but fields with "private" scope are not
	// Local.
	Local nnz.Bool

	// Test is whether this def is defined in test code (as opposed to main
	// code). For example, definitions in Go *_test.go files have Test = true.
	Test bool `json:",omitempty"`

	// Private is whether this definition is private, i.e., if it came from a private repository.
	Private nnz.Bool `json:",omitempty"`

	// Data contains additional language- and toolchain-specific information
	// about the def. Data is used to construct function signatures,
	// import/require statements, language-specific type descriptions, etc.
	Data sqltypes.JSON `json:",omitempty"`
}

// END Def OMIT

var treePathRegexp = regexp.MustCompile(`^(?:[^/]+)(?:/[^/]+)*$`)

func (p TreePath) IsValid() bool {
	return treePathRegexp.MatchString(string(p))
}

func (s *Def) Fmt() DefPrintFormatter { return PrintFormatter(s) }

func (s *Def) sortKey() string { return s.DefKey.String() }

// Propagate describes type/value propagation in code. A Propagate entry from A
// (src) to B (dst) indicates that the type/value of A propagates to B. In Tern,
// this is indicated by A having a "fwd" property whose value is an array that
// includes B.
//
//
// ## Motivation & example
//
// For example, consider the following JavaScript code:
//
//   var a = Foo;
//   var b = a;
//
// Foo, a, and b are each their own def. We could resolve all of them to the
// def of their original type (perhaps Foo), but there are occasions when you
// do want to see only the definition of a or b and examples thereof. Therefore,
// we need to represent them as distinct defs.
//
// Even though Foo, a, and b are distinct defs, there are propagation
// relationships between them that are important to represent. The type of Foo
// propagates to both a and b, and the type of a propagates to b. In this case,
// we would have 3 Propagates: Propagate{Src: "Foo", Dst: "a"}, Propagate{Src:
// "Foo", Dst: "b"}, and Propagate{Src: "a", Dst: "b"}. (The propagation
// relationships could be described by just the first and last Propagates, but
// we explicitly include all paths as a denormalization optimization to avoid
// requiring an unbounded number of DB queries to determine which defs a type
// propagates to or from.)
//
//
// ## Directionality
//
// Propagation is unidirectional, in the general case. In the example above, if
// Foo referred to a JavaScript object and if the code were evaluated, any
// *runtime* type changes (e.g., setting a property) on Foo, a, and b would be
// reflected on all of the others. But this doesn't hold for static analysis;
// it's not always true that if a property "a.x" or "b.x" exists, then "Foo.x"
// exists. The simplest example is when Foo is an external definition. Perhaps
// this example file (which uses Foo as a library) modifies Foo to add a new
// property, but other libraries that use Foo would never see that property
// because they wouldn't be executed in the same context as this example file.
// So, in general, we cannot say that Foo receives all types applied to defs
// that Foo propagates to.
//
//
// ## Hypothetical Python example
//
// Consider the following 2 Python files:
//
//   """file1.py"""
//   class Foo(object): end
//
//   """file2.py"""
//   from .file1 import Foo
//   Foo2 = Foo
//
// In this example, there would be one Propagate: Propagate{Src: "file1/Foo",
// Dst: "file2/Foo2}.
type Propagate struct {
	// Src is the def whose type/value is being propagated to the dst def.
	SrcRepo     string
	SrcPath     DefPath
	SrcUnit     string
	SrcUnitType string

	// Dst is the def that is receiving a propagated type/value from the src def.
	DstRepo     string
	DstPath     DefPath
	DstUnit     string
	DstUnitType string
}

// SQL

func (x DefPath) Value() (driver.Value, error) {
	return string(x), nil
}

func (x *DefPath) Scan(v interface{}) error {
	if data, ok := v.([]byte); ok {
		*x = DefPath(data)
		return nil
	}
	return fmt.Errorf("%T.Scan failed: %v", x, v)
}

func (x TreePath) Value() (driver.Value, error) {
	return string(x), nil
}

func (x *TreePath) Scan(v interface{}) error {
	if data, ok := v.([]byte); ok {
		*x = TreePath(data)
		return nil
	}
	return fmt.Errorf("%T.Scan failed: %v", x, v)
}

// Debugging

func (x Def) String() string {
	s, _ := json.Marshal(x)
	return string(s)
}

// Sorting

type Defs []*Def

func (vs Defs) Len() int           { return len(vs) }
func (vs Defs) Swap(i, j int)      { vs[i], vs[j] = vs[j], vs[i] }
func (vs Defs) Less(i, j int) bool { return vs[i].sortKey() < vs[j].sortKey() }

func (defs Defs) Keys() (keys []DefKey) {
	keys = make([]DefKey, len(defs))
	for i, def := range defs {
		keys[i] = def.DefKey
	}
	return
}

func (defs Defs) KeySet() (keys map[DefKey]struct{}, err error) {
	keys = make(map[DefKey]struct{})
	for _, def := range defs {
		if _, in := keys[def.DefKey]; in {
			return nil, fmt.Errorf("duplicate def key %+v", def.DefKey)
		}
		keys[def.DefKey] = struct{}{}
	}
	return keys, nil
}
