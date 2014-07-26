package src

import (
	"log"

	"github.com/sqs/go-flags"
)

func SetOptionDefaultValue(g *flags.Group, longName string, defaultVal ...string) {
	for _, opt := range g.Options() {
		if opt.LongName == longName {
			opt.Default = defaultVal
			return
		}
	}
	log.Fatalf("Failed to set default value %v for option %q (not found).", defaultVal, longName)
}
