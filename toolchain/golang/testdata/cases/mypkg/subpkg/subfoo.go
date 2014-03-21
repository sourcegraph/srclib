package subpkg

import "mypkg"

type Subqux mypkg.Qux

func subfunc() {
	mypkg.Foo()
}
