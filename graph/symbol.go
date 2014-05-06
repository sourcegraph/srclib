package graph

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"sourcegraph.com/sourcegraph/srcgraph/repo"
)

type (
	SID        int64
	SymbolPath string
	SymbolKind string
)

// SymbolKey specifies a symbol, either concretely or abstractly. A concrete
// symbol key has a non-empty CommitID and refers to a symbol defined in a
// specific commit. An abstract symbol key has an empty CommitID and is
// considered to refer to symbols from any number of commits (so long as the
// Repo, UnitType, Unit, and Path match).
//
// You can think of CommitID as the time dimension. With an empty CommitID, you
// are referring to a symbol that may or may not exist at various times. With a
// non-empty CommitID, you are referring to a specific definition of a symbol at
// the time specified by the CommitID.
type SymbolKey struct {
	// Repo is the VCS repository that defines this symbol. Its Elasticsearch mapping is defined
	// separately.
	Repo repo.URI `json:",omitempty"`

	// CommitID is the ID of the VCS commit that this symbol was defined in. The
	// CommitID is always a full commit ID (40 hexadecimal characters for git
	// and hg), never a branch or tag name.
	CommitID string `db:"commit_id" json:",omitempty"`

	// UnitType is the type name of the source unit (obtained from unit.Type(u))
	// that this symbol was defined in.
	UnitType string `db:"unit_type" json:",omitempty"`

	// Unit is the name of the source unit (obtained from u.Name()) that this
	// symbol was defined in.
	Unit string `json:",omitempty"`

	// Path is the path to this symbol, relative to the repo. Its Elasticsearch mapping is defined
	// separately (because it is a multi_field, which the struct tag can't currently represent).
	Path SymbolPath
}

func (s SymbolKey) String() string {
	b, err := json.Marshal(s)
	if err != nil {
		panic("SymbolKey.String: " + err.Error())
	}
	return string(b)
}

type Symbol struct {
	// SID is a unique, sequential ID for a symbol. It is regenerated each time
	// the symbol is emitted by the grapher and saved to the database. The SID
	// is used as an optimization (e.g., joins are faster on SID than on
	// SymbolKey).
	SID SID `db:"sid" json:",omitempty" elastic:"type:integer,index:no"`

	// SymbolKey is the natural unique key for a symbol. It is stable
	// (subsequent runs of a grapher will emit the same symbols with the same
	// SymbolKeys).
	SymbolKey

	// SpecificPath is the language-specific "path" to this symbol, using
	// language-specific separators (e.g., "::" and "." instead of "/", which is
	// used in the SymbolKey.Path value).
	SpecificPath string `db:"specific_path"`

	// Kind is the language-independent kind of this symbol.
	Kind SymbolKind `elastic:"type:string,index:analyzed"`

	// SpecificKind is the language-specific kind of this symbol (which is in
	// some cases equal to the Kind).
	SpecificKind string `db:"specific_kind"`

	Name string

	// Callable is true if this symbol may be called or invoked, such as in the
	// case of functions or methods.
	Callable bool `db:"callable"`

	File string `elastic:"type:string,index:no"`

	DefStart int `db:"def_start" elastic:"type:integer,index:no"`
	DefEnd   int `db:"def_end" elastic:"type:integer,index:no"`

	Exported bool `elastic:"type:boolean,index:not_analyzed"`

	// Test is whether this symbol is defined in test code (as opposed to main
	// code). For example, definitions in Go *_test.go files have Test = true.
	Test bool `elastic:"type:boolean,index:not_analyzed" json:",omitempty"`

	TypeExpr string `db:"type_expr" json:",omitempty"`
}

func (s *Symbol) Language() string {
	switch s.UnitType {
	case "GoPackage":
		return "Go"
		// TODO(sqs): add Python, JS, etc.
	}
	return "unknown language"
}

// TODO!(sqs): factor this into the individual source unit packages
func (s *Symbol) Signature() string {
	var removeOwnImportPath = func(str string) string {
		if s.UnitType == "GoPackage" {
			return strings.Replace(strings.Replace(str, string(s.Repo)+".", "", -1), string(s.Repo)+"/", "", -1)
		}
		return str
	}

	if s.TypeExpr == "" {
		return ""
	}
	if !s.Callable {
		if (s.Kind == Field || s.Kind == Var || s.Kind == Type) && len(s.TypeExpr) < 50 {
			return " " + removeOwnImportPath(s.TypeExpr)
		}
		return ""
	}

	switch s.UnitType {
	case "GoPackage":
		return removeOwnImportPath(strings.TrimPrefix(s.TypeExpr, "func"))
	case "js":
		return strings.TrimPrefix(s.TypeExpr, "fn")
	case "python":
		// remove up to first paren
		i := strings.Index(s.TypeExpr, "(")
		if i == -1 {
			return s.TypeExpr
		}
		return s.TypeExpr[:i]
	case "ruby":
		return s.TypeExpr
	}
	return s.TypeExpr
}

func (s *Symbol) sortKey() string { return s.SymbolKey.String() }

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
// Foo, a, and b are each their own symbol. We could resolve all of them to the
// symbol of their original type (perhaps Foo), but there are occasions when you
// do want to see only the definition of a or b and examples thereof. Therefore,
// we need to represent them as distinct symbols.
//
// Even though Foo, a, and b are distinct symbols, there are propagation
// relationships between them that are important to represent. The type of Foo
// propagates to both a and b, and the type of a propagates to b. In this case,
// we would have 3 Propagates: Propagate{Src: "Foo", Dst: "a"}, Propagate{Src:
// "Foo", Dst: "b"}, and Propagate{Src: "a", Dst: "b"}. (The propagation
// relationships could be described by just the first and last Propagates, but
// we explicitly include all paths as a denormalization optimization to avoid
// requiring an unbounded number of DB queries to determine which symbols a type
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
// So, in general, we cannot say that Foo receives all types applied to symbols
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
	// Src is the symbol whose type/value is being propagated to the dst symbol.
	SrcRepo     repo.URI
	SrcPath     SymbolPath
	SrcUnit     string
	SrcUnitType string

	// Dst is the symbol that is receiving a propagated type/value from the src symbol.
	DstRepo     repo.URI
	DstPath     SymbolPath
	DstUnit     string
	DstUnitType string
}

const (
	Const   SymbolKind = "const"
	Field              = "field"
	Func               = "func"
	Module             = "module"
	Package            = "package"
	Type               = "type"
	Var                = "var"
)

var AllSymbolKinds = []SymbolKind{Const, Field, Func, Module, Package, Type, Var}

// Returns true iff k is a known symbol kind.
func (k SymbolKind) Valid() bool {
	for _, kk := range AllSymbolKinds {
		if k == kk {
			return true
		}
	}
	return false
}

func KindName(k string) string {
	if strings.HasSuffix(k, "_module") {
		return "module"
	}
	return strings.Replace(k, "_", " ", -1)
}

type ConstData struct {
	ConstValue string
}

// SQL

func (x SID) Value() (driver.Value, error) {
	return int64(x), nil
}

func (x *SID) Scan(v interface{}) error {
	if data, ok := v.(int64); ok {
		*x = SID(data)
		return nil
	}
	return fmt.Errorf("%T.Scan failed: %v", x, v)
}

func (x SymbolPath) Value() (driver.Value, error) {
	return string(x), nil
}

func (x *SymbolPath) Scan(v interface{}) error {
	if data, ok := v.([]byte); ok {
		*x = SymbolPath(data)
		return nil
	}
	return fmt.Errorf("%T.Scan failed: %v", x, v)
}

func (x SymbolKind) Value() (driver.Value, error) {
	return string(x), nil
}

func (x *SymbolKind) Scan(v interface{}) error {
	if data, ok := v.([]byte); ok {
		*x = SymbolKind(data)
		return nil
	}
	return fmt.Errorf("%T.Scan failed: %v", x, v)
}

func (x *ConstData) Value() (driver.Value, error) {
	if x == nil {
		return nil, nil
	}
	return json.Marshal(x)
}

func (x *ConstData) Scan(v interface{}) error {
	if data, ok := v.([]byte); ok {
		return json.Unmarshal(data, x)
	}
	return fmt.Errorf("%T.Scan failed: %v", x, v)
}

// Debugging

func (x Symbol) String() string {
	s, _ := json.Marshal(x)
	return string(s)
}

// Sorting

type Symbols []*Symbol

func (vs Symbols) Len() int           { return len(vs) }
func (vs Symbols) Swap(i, j int)      { vs[i], vs[j] = vs[j], vs[i] }
func (vs Symbols) Less(i, j int) bool { return vs[i].sortKey() < vs[j].sortKey() }

func (syms Symbols) Keys() (keys []SymbolKey) {
	keys = make([]SymbolKey, len(syms))
	for i, sym := range syms {
		keys[i] = sym.SymbolKey
	}
	return
}

func (syms Symbols) SIDs() (ids []SID) {
	ids = make([]SID, len(syms))
	for i, sym := range syms {
		ids[i] = sym.SID
	}
	return
}

func ParseSIDs(sidstrs []string) (sids []SID) {
	sids = make([]SID, len(sidstrs))
	for i, sidstr := range sidstrs {
		sid, err := strconv.Atoi(sidstr)
		if err == nil {
			sids[i] = SID(sid)
		}
	}
	return
}
