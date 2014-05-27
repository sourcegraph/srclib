package graph

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

// // FormatAndSortSymbolsForRepositoryListing uses SymbolFormatters registered by
// // the various toolchains to format and sort symbols for display on the
// // "package" listing page of a repository. The provided symbols slice is sorted
// // in-place.
// func FormatAndSortSymbolsForRepositoryListing(symbols []*Symbol) map[*Symbol]RepositoryListingSymbol {
// 	m := make(map[*Symbol]RepositoryListingSymbol, len(symbols))
// 	for _, s := range symbols {
// 		sf, present := SymbolFormatters[s.UnitType]
// 		if !present {
// 			panic("no SymbolFormatter for symbol with UnitType " + s.UnitType)
// 		}

// 		m[s] = sf.RepositoryListing(s)
// 	}

// 	// sort
// 	ss := &repositoryListingSymbols{m, symbols}
// 	sort.Sort(ss)
// 	return m
// }

// type repositoryListingSymbols struct {
// 	info    map[*Symbol]RepositoryListingSymbol
// 	symbols []*Symbol
// }

// func (s *repositoryListingSymbols) Len() int { return len(s.symbols) }
// func (s *repositoryListingSymbols) Swap(i, j int) {
// 	s.symbols[i], s.symbols[j] = s.symbols[j], s.symbols[i]
// }
// func (s *repositoryListingSymbols) Less(i, j int) bool {
// 	return s.info[s.symbols[i]].SortKey < s.info[s.symbols[j]].SortKey
// }
