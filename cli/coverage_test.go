package cli

import (
	"io/ioutil"
	"testing"
)

func TestStripCode(t *testing.T) {

	source, _ := ioutil.ReadFile("testdata/strip-code/input-source.txt")
	target, _ := ioutil.ReadFile("testdata/strip-code/expected-source.txt")
	expected := string(target)

	actual := string(stripCode(source))
	if actual != expected {
		t.Errorf("got\n%v\n, want\n%v\n", actual, expected)
	}

	source = []byte("abc\n//def\nfgh")
	expected = "abc\n\nfgh"

	actual = string(stripCode(source))
	if actual != expected {
		t.Errorf("got\n%v\n, want\n%v\n", actual, expected)
	}

}

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
		{
			"",
			0,
		},
	}

	for _, test := range tests {
		actual := numLines([]byte(test.data))
		if actual != test.expected {
			t.Errorf("%s, got %v, want %v", test.data, actual, test.expected)
		}
	}

}
