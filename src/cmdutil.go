package src

import (
	"fmt"
	"log"

	"github.com/sqs/go-flags"
)

// MarshalArgs takes a go-flags flags.Group and turns it into an equivalent
// []string for use as command-line args.
func MarshalArgs(group *flags.Group) ([]string, error) {
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

func SetOptionDefaultValue(g *flags.Group, longName string, defaultVal ...string) {
	for _, opt := range g.Options() {
		if opt.LongName == longName {
			opt.Default = defaultVal
			return
		}
	}
	log.Fatalf("Failed to set default value %v for option %q (not found).", defaultVal, longName)
}
