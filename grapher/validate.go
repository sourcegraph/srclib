package grapher

import (
	"fmt"

	"strings"

	"sourcegraph.com/sourcegraph/srclib/graph"
)

func ValidateRefs(refs []*graph.Ref) (errs MultiError) {
	refKeys := make(map[graph.RefKey]struct{})
	for _, ref := range refs {
		key := ref.RefKey()
		if _, in := refKeys[key]; in {
			errs = append(errs, fmt.Errorf("duplicate ref key: %+v", key))
		} else {
			refKeys[key] = struct{}{}
		}
	}
	return
}

func ValidateDefs(defs []*graph.Def) (errs MultiError) {
	defKeys := make(map[graph.DefKey]struct{})
	for _, def := range defs {
		key := def.DefKey
		if _, in := defKeys[key]; in {
			errs = append(errs, fmt.Errorf("duplicate def key: %+v", key))
		} else {
			defKeys[key] = struct{}{}
		}
	}
	return
}

type MultiError []error

func (e MultiError) Error() string {
	var msgs []string
	for _, err := range e {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "\n")
}
