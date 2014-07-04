package ruby

import "sourcegraph.com/sourcegraph/srcgraph/unit"

const rubyLibUnitType = "ruby"

func init() {
	unit.Register(rubyLibUnitType, &RubyLib{})
}

// RubyLib is a collection of Ruby source files that are not in a gem. It is
// used for the Ruby stdlib (and it will probably be useful for Ruby non-gem
// apps).
type RubyLib struct {
	LibName string
	Dir     string
	Files   []string
}

func (p RubyLib) Name() string    { return p.LibName }
func (p RubyLib) RootDir() string { return p.Dir }
func (p RubyLib) Paths() []string { return p.Files }
