package cli

import (
	"testing"
)

func TestNumLines(t *testing.T) {
	tests := []struct {
		data     string
		expected int
	}{
		{
			"do\ncats\neat\nbats",
			4,
		},
		{
			"do\n\n\n\ncats\neat\nbats",
			4,
		},
		{
			"do\r\n\r\n\r\ncats\neat\nbats",
			4,
		},
		{
			"",
			0,
		},
		{
			"abc\n//def\nfgh",
			2,
		},
	}

	for _, test := range tests {
		actual := numLines([]byte(test.data))
		if actual != test.expected {
			t.Errorf("%s, got %v, want %v", test.data, actual, test.expected)
		}
	}

}
