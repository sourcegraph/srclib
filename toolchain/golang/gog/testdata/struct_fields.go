package testdata

type A struct {
	x string
}

func t1() {
	var a A
	_ = a.x
}
