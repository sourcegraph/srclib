package toolchain

type Toolchain interface {
}

var Toolchains = make(map[string]Toolchain)

// Register makes a toolchain available by the provided name. If Register is
// called twice with the same name or if toolchain is nil, it panics
func Register(name string, toolchain Toolchain) {
	if _, dup := Toolchains[name]; dup {
		panic("toolchain: Register called twice for driver " + name)
	}
	if toolchain == nil {
		panic("toolchain: Register toolchain is nil")
	}
	Toolchains[name] = toolchain
}
