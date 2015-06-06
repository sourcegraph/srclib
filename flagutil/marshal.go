package flagutil

import (
	"fmt"
	"strings"

	"sourcegraph.com/sourcegraph/go-flags"
)

// MarshalArgs takes a struct with go-flags field tags and turns it into an
// equivalent []string for use as command-line args.
func MarshalArgs(v interface{}) ([]string, error) {
	parser := flags.NewParser(nil, flags.None)
	group, err := parser.AddGroup("", "", v)
	if err != nil {
		return nil, err
	}
	return marshalArgsInGroup(group, "")
}

func marshalArgsInGroup(group *flags.Group, prefix string) ([]string, error) {
	var args []string
	for _, opt := range group.Options() {
		flagStr := opt.String()

		// handle flags with both short and long (just get the long)
		if i := strings.Index(flagStr, ", --"); i != -1 {
			flagStr = flagStr[i+2:]
		}

		switch v := opt.Value().(type) {
		case flags.Marshaler:
			s, err := v.MarshalFlag()
			if err != nil {
				return nil, err
			}
			args = append(args, flagStr, s)
		case []string:
			for _, s := range v {
				args = append(args, flagStr, s)
			}
		case bool:
			if v {
				args = append(args, flagStr)
			}
		default:
			args = append(args, flagStr, fmt.Sprintf("%v", v))
		}
	}
	for _, g := range group.Groups() {
		// TODO(sqs): assumes that the NamespaceDelimiter is "."
		const namespaceDelimiter = "."
		groupArgs, err := marshalArgsInGroup(g, g.Namespace+namespaceDelimiter)
		if err != nil {
			return nil, err
		}
		args = append(args, groupArgs...)
	}
	return args, nil
}
