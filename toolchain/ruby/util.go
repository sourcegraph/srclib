package ruby

import (
	"path/filepath"
	"strings"
)

var StdlibGemNameSentinel = "<<RUBY_STDLIB>>"

// getGemNameFromGemYardocFile converts a path of the form
// "/tmp/grapher-output-test308543943/github.com-ruth-my_ruby_gem/ruby/gems-./ruby/2.0.0/gems/sample_ruby_gem-0.0.1/.yardoc"
// to "sample_ruby_gem"
func getGemNameFromGemYardocFile(gemYardocFile string) string {
	if gemYardocFile == "" {
		return ""
	}
	if gemYardocFile == RubyStdlibYARDocDir {
		return StdlibGemNameSentinel
	}
	nameVer := filepath.Base(filepath.Dir(gemYardocFile))
	i := strings.LastIndex(nameVer, "-")
	if i == -1 {
		return nameVer
	}
	return nameVer[:i]
}
