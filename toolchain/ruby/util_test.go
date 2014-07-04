package ruby

import (
	"testing"
)

func TestGetGemNameFromGemYardocFile(t *testing.T) {
	tests := []struct {
		gemYardocFile string
		want          string
	}{
		{"/tmp/grapher-output-test308543943/github.com-ruth-my_ruby_gem/ruby/gems-./ruby/2.0.0/gems/sample_ruby_gem-0.0.1/.yardoc", "sample_ruby_gem"},
		{"", ""},
		{"/tmp/grapher-output-test308543943/github.com-ruth-my_ruby_gem/ruby/gems-./ruby/2.0.0/gems/sample_ruby_gem-with-multiple-dashes-0.0.1/.yardoc", "sample_ruby_gem-with-multiple-dashes"},
		{RubyStdlibYARDocDir, StdlibGemNameSentinel},
	}
	for _, test := range tests {
		got := getGemNameFromGemYardocFile(test.gemYardocFile)
		if test.want != got {
			t.Errorf("%s: want %q, got %q", test.gemYardocFile, test.want, got)
		}
	}
}
