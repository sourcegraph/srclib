package srclib

import (
	"errors"
	"strings"
)

func (t *ToolRef) UnmarshalFlag(value string) error {
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return errors.New("expected format 'TOOLCHAIN:TOOL' (separated by 1 colon)")
	}
	t.Toolchain = parts[0]
	t.Subcmd = parts[1]
	return nil
}

func (t ToolRef) MarshalFlag() (string, error) {
	return t.Toolchain + ":" + t.Subcmd, nil
}
