package src

import (
	"log"
	"os"
	"path/filepath"

	"github.com/sqs/go-flags"
)

// Directory is flags.Completer that provides directory name completion.
//
// TODO(sqs): this is annoying. it only completes the dir name and doesn't let
// you keep typing the arg.
type Directory string

// Complete implements flags.Completer and returns a list of existing
// directories with the given prefix.
func (d Directory) Complete(match string) []flags.Completion {
	names, err := filepath.Glob(match + "*")
	if err != nil {
		log.Println(err)
		return nil
	}

	var dirs []flags.Completion
	for _, name := range names {
		if fi, err := os.Stat(name); err == nil && fi.Mode().IsDir() {
			dirs = append(dirs, flags.Completion{Item: name + "/"})
		}
	}
	return dirs
}
