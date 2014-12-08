package flagutil

import (
	"fmt"
	"strings"

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
		flagStr := opt.String()

		// handle flags with both short and long (just get the long)
		if i := strings.Index(flagStr, ", --"); i != -1 {
			flagStr = flagStr[i+2:]
		}

		v := opt.Value()
		if m, ok := v.(flags.Marshaler); ok {
			s, err := m.MarshalFlag()
			if err != nil {
				return nil, err
			}
			args = append(args, flagStr, s)
		} else if ss, ok := v.([]string); ok {
			for _, s := range ss {
				args = append(args, flagStr, s)
			}
		} else if bv, ok := v.(bool); ok {
			if bv {
				args = append(args, flagStr)
			}
		} else {
			args = append(args, flagStr, fmt.Sprintf("%v", opt.Value()))
		}
	}
	return args, nil
}
