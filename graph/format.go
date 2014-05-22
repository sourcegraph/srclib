package graph

import "sort"

var SymbolFormatters = make(map[string]SymbolFormatter)

// RegisterSymbolFormatter makes a SymbolFormatter available for symbols with
// the specified unitType. If Register is called twice with the same unitType or
// if sf is nil, it panics
func RegisterSymbolFormatter(unitType string, sf SymbolFormatter) {
	if _, dup := SymbolFormatters[unitType]; dup {
		panic("graph: RegisterSymbolFormatter called twice for unit type " + unitType)
	}
	if sf == nil {
		panic("graph: RegisterSymbolFormatter toolchain is nil")
	}
	SymbolFormatters[unitType] = sf
}

// SymbolFormatter formats symbols for display in various places.
type SymbolFormatter interface {
	// RepositoryListing formats a symbol to be displayed on the "package"
	// listing page of a repository.
	RepositoryListing(pkglike *Symbol) RepositoryListingSymbol

	// QualifiedName renders the qualified name of s. If base is set, the name
	// is rendered as s would be referenced in the scope of base. If base is
	// nil, the name is rendered as s would be referenced in the outermost scope
	// of the source unit.
	//
	// TODO(sqs): The innerLinks param determines whether the returned
	// HTML contains <a href="..."> links pointing to all other symbols
	// referenced in the result (for example, in the qualified name for a Go
	// method, such as `(*MyType).MyMethod`, both MyType and MyMethod are
	// included in the resulting string).
	//
	// The HTML must include at least one non-empty HTML element with a class
	// "name", which is the name of the symbol itself.
	//
	// For example, the qualified name for a Go func Foo(), with relativeTo == nil
	// (i.e., from Go package scope), is `<a href="..." class="name">Foo</a>`.
	QualifiedName(s *Symbol, relativeTo *SymbolKey) string

	// TypeString is the type string of the symbol in source code.
	//
	// In general, the returned string follows these rules:
	//
	// * s is a function: `(arg1, arg2, ..., argN)` with language-specific type
	//   annotations.
	//
	// * s is a type: either a short definition or empty (don't include the
	//   full struct definition, for example).
	//
	// * s is a variable: the type name
	//
	// * s is a syntactic construct with no type information, such as a package
	//   or module: empty
	TypeString(s *Symbol) string

	// LanguageName is the name of the programming language that s is in; e.g.,
	// "Python" or "Go".
	LanguageName(s *Symbol) string

	// KindName is the language-speciifc name of the symbol's kind, but not
	// including the language (which can be obtained using LanguageName).
	KindName(s *Symbol) string
}

// RepositoryListingSymbol holds rendered display text to show on the "package"
// listing page of a repository.
type RepositoryListingSymbol struct {
	// Name is the full name shown on the page.
	Name string

	// NameLabel is a label displayed next to the Name, such as "(main package)"
	// to denote that a package is a Go main package.
	NameLabel string

	// Language is the source language of the symbol, with any additional
	// specifiers, such as "JavaScript (node.js)".
	Language string

	// SortKey is the key used to lexicographically sort all of the symbols on
	// the page.
	SortKey string
}

// FormatAndSortSymbolsForRepositoryListing uses SymbolFormatters registered by
// the various toolchains to format and sort symbols for display on the
// "package" listing page of a repository. The provided symbols slice is sorted
// in-place.
func FormatAndSortSymbolsForRepositoryListing(symbols []*Symbol) map[*Symbol]RepositoryListingSymbol {
	m := make(map[*Symbol]RepositoryListingSymbol, len(symbols))
	for _, s := range symbols {
		sf, present := SymbolFormatters[s.UnitType]
		if !present {
			panic("no SymbolFormatter for symbol with UnitType " + s.UnitType)
		}

		m[s] = sf.RepositoryListing(s)
	}

	// sort
	ss := &repositoryListingSymbols{m, symbols}
	sort.Sort(ss)
	return m
}

type repositoryListingSymbols struct {
	info    map[*Symbol]RepositoryListingSymbol
	symbols []*Symbol
}

func (s *repositoryListingSymbols) Len() int { return len(s.symbols) }
func (s *repositoryListingSymbols) Swap(i, j int) {
	s.symbols[i], s.symbols[j] = s.symbols[j], s.symbols[i]
}
func (s *repositoryListingSymbols) Less(i, j int) bool {
	return s.info[s.symbols[i]].SortKey < s.info[s.symbols[j]].SortKey
}
