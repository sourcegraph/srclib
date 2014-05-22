package graph

import (
	"reflect"
	"testing"
)

type testSymbolFormatter struct{}

func (f testSymbolFormatter) RepositoryListing(s *Symbol) RepositoryListingSymbol {
	return RepositoryListingSymbol{
		Name:    s.Name,
		SortKey: s.Name,
	}
}

func TestFormatAndSortSymbolsForRepositoryListing(t *testing.T) {
	RegisterSymbolFormatter("t", testSymbolFormatter{})
	defer func() {
		SymbolFormatters = nil
	}()

	symbols := []*Symbol{
		{SymbolKey: SymbolKey{UnitType: "t"}, Name: "z"},
		{SymbolKey: SymbolKey{UnitType: "t"}, Name: "a"},
	}

	want := map[*Symbol]RepositoryListingSymbol{
		symbols[0]: RepositoryListingSymbol{Name: "z", NameLabel: "", Language: "", SortKey: "z"},
		symbols[1]: RepositoryListingSymbol{Name: "a", NameLabel: "", Language: "", SortKey: "a"},
	}

	fmtSymbols := FormatAndSortSymbolsForRepositoryListing(symbols)

	// Check that fmtSymbols is sorted (was [z,a], should be [a,z]).
	if s1 := symbols[0]; s1.Name != "a" {
		t.Errorf("got sorted symbol1 name %q, want 'a'", s1.Name)
	}
	if s2 := symbols[1]; s2.Name != "z" {
		t.Errorf("got sorted symbol2 name %q, want 'z'", s2.Name)
	}

	if !reflect.DeepEqual(fmtSymbols, want) {
		t.Errorf("got formatted symbols map %+v, want %+v", fmtSymbols, want)
	}
}
