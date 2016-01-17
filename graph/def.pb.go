// Code generated by protoc-gen-gogo.
// source: def.proto
// DO NOT EDIT!

/*
Package graph is a generated protocol buffer package.

It is generated from these files:
	def.proto
	doc.proto
	output.proto
	ref.proto

It has these top-level messages:
	DefKey
	Def
	DefDoc
	DefFormatStrings
	QualFormatStrings
	Doc
	Output
	Ref
	RefDefKey
*/
package graph

import proto "github.com/gogo/protobuf/proto"
import fmt "fmt"
import math "math"

// discarding unused import gogoproto "github.com/gogo/protobuf/gogoproto"

import sourcegraph_com_sqs_pbtypes "sourcegraph.com/sqs/pbtypes"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

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
	// Repo is the VCS repository that defines this definition.
	Repo string `protobuf:"bytes,1,opt,name=Repo,proto3" json:"Repo,omitempty"`
	// CommitID is the ID of the VCS commit that this definition was defined in. The
	// CommitID is always a full commit ID (40 hexadecimal characters for git
	// and hg), never a branch or tag name.
	CommitID string `protobuf:"bytes,2,opt,name=CommitID,proto3" json:"CommitID,omitempty"`
	// UnitType is the type name of the source unit (obtained from unit.Type(u))
	// that this definition was defined in.
	UnitType string `protobuf:"bytes,3,opt,name=UnitType,proto3" json:"UnitType,omitempty"`
	// Unit is the name of the source unit (obtained from u.Name()) that this
	// definition was defined in.
	Unit string `protobuf:"bytes,4,opt,name=Unit,proto3" json:"Unit,omitempty"`
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
	Path string `protobuf:"bytes,5,opt,name=Path,proto3" json:"Path"`
}

func (m *DefKey) Reset()         { *m = DefKey{} }
func (m *DefKey) String() string { return proto.CompactTextString(m) }
func (*DefKey) ProtoMessage()    {}

// Def is a definition in code.
type Def struct {
	// DefKey is the natural unique key for a def. It is stable
	// (subsequent runs of a grapher will emit the same defs with the same
	// DefKeys).
	DefKey `protobuf:"bytes,1,opt,name=Key,embedded=Key" json:""`
	// Name of the definition. This need not be unique.
	Name string `protobuf:"bytes,2,opt,name=Name,proto3" json:"Name"`
	// Kind is the kind of thing this definition is. This is
	// language-specific. Possible values include "type", "func",
	// "var", etc.
	Kind     string `protobuf:"bytes,3,opt,name=Kind,proto3" json:"Kind,omitempty"`
	File     string `protobuf:"bytes,4,opt,name=File,proto3" json:"File"`
	DefStart uint32 `protobuf:"varint,5,opt,name=DefStart,proto3" json:"DefStart"`
	DefEnd   uint32 `protobuf:"varint,6,opt,name=DefEnd,proto3" json:"DefEnd"`
	// Exported is whether this def is part of a source unit's
	// public API. For example, in Java a "public" field is
	// Exported.
	Exported bool `protobuf:"varint,7,opt,name=Exported,proto3" json:"Exported,omitempty"`
	// Local is whether this def is local to a function or some
	// other inner scope. Local defs do *not* have module,
	// package, or file scope. For example, in Java a function's
	// args are Local, but fields with "private" scope are not
	// Local.
	Local bool `protobuf:"varint,8,opt,name=Local,proto3" json:"Local,omitempty"`
	// Test is whether this def is defined in test code (as opposed to main
	// code). For example, definitions in Go *_test.go files have Test = true.
	Test bool `protobuf:"varint,9,opt,name=Test,proto3" json:"Test,omitempty"`
	// Data contains additional language- and toolchain-specific information
	// about the def. Data is used to construct function signatures,
	// import/require statements, language-specific type descriptions, etc.
	Data sourcegraph_com_sqs_pbtypes.RawMessage `protobuf:"bytes,10,opt,name=Data,proto3,customtype=sourcegraph.com/sqs/pbtypes.RawMessage" json:"Data,omitempty"`
	// Docs are docstrings for this Def. This field is not set in the
	// Defs produced by graphers; they should emit docs in the
	// separate Docs field on the graph.Output struct.
	Docs []*DefDoc `protobuf:"bytes,11,rep,name=Docs" json:"Docs,omitempty"`
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
	TreePath string `protobuf:"bytes,17,opt,name=TreePath,proto3" json:"TreePath,omitempty"`
}

func (m *Def) Reset()         { *m = Def{} }
func (m *Def) String() string { return proto.CompactTextString(m) }
func (*Def) ProtoMessage()    {}

// DefDoc is documentation on a Def.
type DefDoc struct {
	// Format is the the MIME-type that the documentation is stored
	// in. Valid formats include 'text/html', 'text/plain',
	// 'text/x-markdown', text/x-rst'.
	Format string `protobuf:"bytes,1,opt,name=Format,proto3" json:"Format"`
	// Data is the actual documentation text.
	Data string `protobuf:"bytes,2,opt,name=Data,proto3" json:"Data"`
}

func (m *DefDoc) Reset()         { *m = DefDoc{} }
func (m *DefDoc) String() string { return proto.CompactTextString(m) }
func (*DefDoc) ProtoMessage()    {}

// DefFormatStrings contains the various def format strings.
type DefFormatStrings struct {
	Name                 QualFormatStrings `protobuf:"bytes,1,opt,name=Name" json:"Name"`
	Type                 QualFormatStrings `protobuf:"bytes,2,opt,name=Type" json:"Type"`
	NameAndTypeSeparator string            `protobuf:"bytes,3,opt,name=NameAndTypeSeparator,proto3" json:"NameAndTypeSeparator,omitempty"`
	Language             string            `protobuf:"bytes,4,opt,name=Language,proto3" json:"Language,omitempty"`
	DefKeyword           string            `protobuf:"bytes,5,opt,name=DefKeyword,proto3" json:"DefKeyword,omitempty"`
	Kind                 string            `protobuf:"bytes,6,opt,name=Kind,proto3" json:"Kind,omitempty"`
}

func (m *DefFormatStrings) Reset()         { *m = DefFormatStrings{} }
func (m *DefFormatStrings) String() string { return proto.CompactTextString(m) }
func (*DefFormatStrings) ProtoMessage()    {}

// QualFormatStrings holds the formatted string for each Qualification for a def
// (for either Name or Type).
type QualFormatStrings struct {
	Unqualified             string `protobuf:"bytes,1,opt,name=Unqualified,proto3" json:"Unqualified,omitempty"`
	ScopeQualified          string `protobuf:"bytes,2,opt,name=ScopeQualified,proto3" json:"ScopeQualified,omitempty"`
	DepQualified            string `protobuf:"bytes,3,opt,name=DepQualified,proto3" json:"DepQualified,omitempty"`
	RepositoryWideQualified string `protobuf:"bytes,4,opt,name=RepositoryWideQualified,proto3" json:"RepositoryWideQualified,omitempty"`
	LanguageWideQualified   string `protobuf:"bytes,5,opt,name=LanguageWideQualified,proto3" json:"LanguageWideQualified,omitempty"`
}

func (m *QualFormatStrings) Reset()         { *m = QualFormatStrings{} }
func (m *QualFormatStrings) String() string { return proto.CompactTextString(m) }
func (*QualFormatStrings) ProtoMessage()    {}
