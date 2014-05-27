package graph

import (
	"fmt"
	"testing"
)

type testFormatter struct{}

func (_ testFormatter) Name(qual Qualification) string {
	switch qual {
	case Unqualified:
		return "name"
	case ScopeQualified:
		return "scope.name"
	case DepQualified:
		return "imp.scope.name"
	case LanguageWideQualified:
		return "lib.scope.name"
	}
	panic("Name: unrecognized Qualification: " + fmt.Sprint(qual))
}

func (_ testFormatter) Type(qual Qualification) string {
	switch qual {
	case Unqualified:
		return "typeName"
	case ScopeQualified:
		return "scope.typeName"
	case DepQualified:
		return "imp.scope.typeName"
	case LanguageWideQualified:
		return "lib.scope.typeName"
	}
	panic("Type: unrecognized Qualification: " + fmt.Sprint(qual))
}

func (_ testFormatter) Language() string             { return "lang" }
func (_ testFormatter) DefKeyword() string           { return "defkw" }
func (_ testFormatter) NameAndTypeSeparator() string { return "_" }
func (_ testFormatter) Kind() string                 { return "kind" }

func TestPrintFormatter(t *testing.T) {
	const unitType = "TestFormatter"
	RegisterMakeSymbolFormatter("TestFormatter", func(*Symbol) SymbolFormatter { return testFormatter{} })
	symbol := &Symbol{SymbolKey: SymbolKey{UnitType: unitType}}
	tests := []struct {
		format string
		want   string
	}{
		{"%n", "name"},
		{"%.0n", "name"},
		{"%.1n", "scope.name"},
		{"%.2n", "imp.scope.name"},
		{"%.3n", "lib.scope.name"},
		{"%t", "typeName"},
		{"%.0t", "typeName"},
		{"%.1t", "scope.typeName"},
		{"%.2t", "imp.scope.typeName"},
		{"%.3t", "lib.scope.typeName"},
		{"% t", "_typeName"},
		{"%w", "defkw"},
		{"%k", "kind"},
	}
	for _, test := range tests {
		str := fmt.Sprintf(test.format, PrintFormatter(symbol))
		if str != test.want {
			t.Errorf("Sprintf(%q, symbol): got %q, want %q", test.format, str, test.want)
		}
	}
}
