package testdata

type Foo struct {
	*Foo
}

func (f *Foo) Bar(baz string, qux ...int) (a int, x *struct{ *Foo }) {
	if baz, foo := 1, 2; baz != foo {
		return 1, nil
	}
	_ = x.Foo
	_ = a
	return
}
