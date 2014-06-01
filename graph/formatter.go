package graph

import "fmt"

// A Qualification specifies how much to qualify names when formatting symbols
// and their type information.
type Qualification string

const (
	// An Unqualified name is just the symbol's name.
	//
	// Examples:
	//
	//   Go method         `MyMethod`
	//   Python method     `my_method`
	//   JavaScript method `myMethod`
	Unqualified Qualification = ""

	// A ScopeQualified name is the language-specific description of the
	// symbol's defining scope plus the symbol's unqualified name. It should
	// uniquely describe the symbol among all other symbols defined in the same
	// logical package (but this is not strictly defined or enforced).
	//
	// Examples:
	//
	//   Go method         `(*MyType).MyMethod`
	//   Python method     `MyClass.my_method`
	//   JavaScript method `MyConstructor.prototype.myMethod`
	ScopeQualified = "scope"

	// A DepQualified name is the package/module name (as seen by an external
	// library that imports/depends on the symbol's package/module) plus the
	// symbol's scope-qualified name. If there are nested packages, it should
	// describe enough of the package hierarchy to distinguish it from other
	// similarly named symbols (but this is not strictly defined or enforced).
	//
	// Examples:
	//
	//   Go method       `(*mypkg.MyType).MyMethod`
	//   Python method   `mypkg.MyClass.my_method`
	//   CommonJS method `mymodule.MyConstructor.prototype.myMethod`
	DepQualified = "dep"

	// A RepositoryWideQualified name is the full package/module name(s) plus
	// the symbol's scope-qualified name. It should describe enough of the
	// package hierarchy so that it is unique in its repository.
	// RepositoryWideQualified differs from DepQualified in that the former
	// includes the full nested package/module path from the repository root
	// (e.g., 'a/b.C' for a Go func C in the repository 'github.com/user/a'
	// subdirectory 'b'), while DepQualified would only be the last directory
	// component (e.g., 'b.C' in that example).
	//
	// Examples:
	//
	//   Go method       `(*mypkg/subpkg.MyType).MyMethod`
	//   Python method   `mypkg.subpkg.MyClass.my_method` (unless mypkg =~ subpkg)
	//   CommonJS method `mypkg.mymodule.MyConstructor.prototype.myMethod` (unless mypkg =~ mymodule)
	RepositoryWideQualified = "repo-wide"

	// A LanguageWideQualified name is the library/repository name plus the
	// package-qualified symbol name. It should describe the symbol so that it
	// is logically unique among all symbols that could reasonably exist for the
	// language that the symbol is written in (but this is not strictly defined
	// or enforced).
	//
	// Examples:
	//
	//   Go method       `(*github.com/user/repo/mypkg.MyType).MyMethod`
	//   Python method   `mylib.MyClass.my_method` (if mylib =~ mypkg, as for Django, etc.)
	//   CommonJS method `mylib.MyConstructor.prototype.myMethod` (if mylib =~ mymod, as for caolan/async, etc.)
	LanguageWideQualified = "lang-wide"
)

// qualLevels associates a number (the slice index) with each Qualification, for
// use in format strings (so that, e.g., "%.0n" means Unqualified name and
// "%.2n" means DepQualified name).
var qualLevels = []Qualification{
	Unqualified, ScopeQualified, DepQualified, RepositoryWideQualified, LanguageWideQualified,
}

// A MakeSymbolFormatter is a function, typically implemented by toolchains,
// that creates a SymbolFormatter for a symbol.
type MakeSymbolFormatter func(*Symbol) SymbolFormatter

// MakeSymbolFormatter holds MakeSymbolFormatters that toolchains have
// registered with RegisterMakeSymbolFormatter.
var MakeSymbolFormatters = make(map[string]MakeSymbolFormatter)

// RegisterMakeSymbolFormatter makes a SymbolFormatter constructor function
// (MakeSymbolFormatter) available for symbols with the specified unitType. If
// Register is called twice with the same unitType or if sf is nil, it panics
func RegisterMakeSymbolFormatter(unitType string, f MakeSymbolFormatter) {
	if _, dup := MakeSymbolFormatters[unitType]; dup {
		panic("graph: RegisterMakeSymbolFormatter called twice for unit type " + unitType)
	}
	if f == nil {
		panic("graph: RegisterMakeSymbolFormatter toolchain is nil")
	}
	MakeSymbolFormatters[unitType] = f
}

// SymbolFormatter formats a symbol.
type SymbolFormatter interface {
	// Name formats the symbol's name with the specified level of qualification.
	Name(qual Qualification) string

	// Type is the type of the symbol s, if s is not itself a type. If s is
	// itself a type, then Type returns its underlying type.
	//
	// Outputs:
	//
	//   TYPE OF s          RESULT
	//   ------------   -----------------------------------------------------------------
	//   named type     the named type's name
	//   primitive      the primitive's name
	//   function       `(arg1, arg2, ..., argN)` with language-specific type annotations
	//   package        empty
	//   anon. type     the leading keyword (or similar) of the anonymous type definition
	//
	// These rules are not strictly defined or enforced. Language toolchains
	// should freely bend the rules (after noting important exceptions here) to
	// produce sensible output.
	Type(qual Qualification) string

	// NameAndTypeSeparator is the string that should be inserted between the
	// symbol's name and type. This is typically empty for functions (so that
	// they are formatted with the left paren immediately following the name,
	// like `F(a)`) and a single space for other symbols (e.g., `MyVar string`).
	NameAndTypeSeparator() string

	// Language is the name of the programming language that s is in; e.g.,
	// "Python" or "Go".
	Language() string

	// DefKeyword is the language keyword used to define the symbol (e.g.,
	// 'class', 'type', 'func').
	DefKeyword() string

	// Kind is the language-specific kind of this symbol (e.g., 'package', 'field', 'CommonJS module').
	Kind() string
}

// Formatter creates a string formatter for a symbol.
//
// The verbs:
//
//   %n     qualified name
//   %w     language keyword used to define the symbol (e.g., 'class', 'type', 'func')
//   %k     language-specific kind of this symbol (e.g., 'package', 'field', 'CommonJS module')
//   %t     type
//
// The flags:
//   ' '    (in `% t`) prepend the language-specific delimiter between a symbol's name and type
//
// See SymbolFormatter for more information.
func PrintFormatter(s *Symbol) SymbolPrintFormatter {
	mk, ok := MakeSymbolFormatters[s.UnitType]
	if !ok {
		panic("PrintFormatter: no formatter for unit type " + s.UnitType)
	}
	sf := mk(s)
	if sf == nil {
		panic("PrintFormatter: nil SymbolFormatter")
	}
	return &printFormatter{sf}
}

type printFormatter struct{ SymbolFormatter }

func (pf *printFormatter) Format(f fmt.State, c rune) {
	var qual Qualification
	if prec, ok := f.Precision(); ok {
		if prec < 0 || prec >= len(qualLevels) {
			fmt.Fprint(f, "%%!%c(invalid qual %d)", c, prec)
			return
		}
		qual = qualLevels[prec]
	}

	switch c {
	case 'n':
		fmt.Fprint(f, pf.Name(qual))
	case 'w':
		fmt.Fprint(f, pf.DefKeyword())
	case 'k':
		fmt.Fprint(f, pf.Kind())
	case 't':
		if f.Flag(' ') {
			fmt.Fprint(f, pf.NameAndTypeSeparator())
		}
		fmt.Fprint(f, pf.Type(qual))
	}
}

type SymbolPrintFormatter interface {
	SymbolFormatter
	fmt.Formatter
}
