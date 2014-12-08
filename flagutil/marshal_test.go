package flagutil

import (
	"reflect"
	"testing"
)

func TestMarshalArgs(t *testing.T) {
	tests := []struct {
		group    interface{}
		wantArgs []string
	}{
		{
			group: &struct {
				Foo bool `short:"f"`
				Bar bool `short:"b"`
			}{Foo: true},
			wantArgs: []string{"-f"},
		},
		{
			group: &struct {
				Foo string `short:"f"`
			}{Foo: "bar"},
			wantArgs: []string{"-f", "bar"},
		},
		{
			group: &struct {
				Foo string `long:"foo"`
			}{Foo: "bar"},
			wantArgs: []string{"--foo", "bar"},
		},
		{
			group: &struct {
				Foo string `short:"f"`
			}{Foo: "bar baz"},
			wantArgs: []string{"-f", "bar baz"},
		},
		{
			group: &struct {
				Foo string `long:"foo"`
			}{Foo: "bar baz"},
			wantArgs: []string{"--foo", "bar baz"},
		},
		{
			group: &struct {
				Foo string `short:"f" long:"foo"`
			}{Foo: "bar"},
			wantArgs: []string{"--foo", "bar"},
		},
		{
			group: &struct {
				Foo []string `long:"foo"`
			}{Foo: []string{"bar", "baz"}},
			wantArgs: []string{"--foo", "bar", "--foo", "baz"},
		},
		{
			group: &struct {
				Foo []string `long:"foo"`
			}{Foo: []string{}},
			wantArgs: nil,
		},
	}
	for _, test := range tests {
		args, err := MarshalArgs(test.group)
		if err != nil {
			t.Error(err)
			continue
		}

		if !reflect.DeepEqual(args, test.wantArgs) {
			t.Errorf("got args %v, want %v", args, test.wantArgs)
		}
	}
}
