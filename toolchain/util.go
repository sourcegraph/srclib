package toolchain

import (
	"fmt"

	"github.com/sqs/go-flags"
)

// MarshalArgs takes a struct with go-flags field tags and turns it into an
// equivalent []string for use as command-line args.
func MarshalArgs(v interface{}) ([]string, error) {
	parser := flags.NewParser(nil, flags.None)
	group, err := parser.AddGroup("", "", v)
	if err != nil {
		return nil, err
	}

	var args []string
	for _, opt := range group.Options() {
		args = append(args, opt.String())

		v := opt.Value()
		if m, ok := v.(flags.Marshaler); ok {
			s, err := m.MarshalFlag()
			if err != nil {
				return nil, err
			}
			args = append(args, s)
		} else {
			args = append(args, fmt.Sprintf("%v", opt.Value()))
		}
	}
	return args, nil
}
