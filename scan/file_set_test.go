package scan

import (
	"reflect"
	"testing"
)

func TestFileSet_FilesToAnalyze(t *testing.T) {
	u := &FileSet{Files: map[string]*FileInfo{
		"foo": {Analyze: true}, "bar": {Analyze: false},
	}}
	got := u.FilesToAnalyze()
	want := []string{"foo"}
	if !reflect.DeepEqual(want, got) {
		t.Errorf("want %v, got %v", want, got)
	}
}

func TestFileSet_FilesToBlame(t *testing.T) {
	u := &FileSet{Files: map[string]*FileInfo{
		"foo": {Blame: true}, "bar": {Blame: false},
	}}
	got := u.FilesToBlame()
	want := []string{"foo"}
	if !reflect.DeepEqual(want, got) {
		t.Errorf("want %v, got %v", want, got)
	}
}
