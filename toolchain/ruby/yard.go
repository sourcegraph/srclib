// +build ignore

package ruby

import (
	"encoding/json"
)

func (g *rootGrapher) yardObjects(files []string) ([]*rubyObject, error) {
	args := []string{"condense"}
	args = append(args, files...)
	cmd := rvmCommand(YARDPath, args...)
	cmd.Dir = g.ctx.Dir
	//	g.ctx.Log.Printf("Running: %v", cmd.Args)
	cmd.Stderr = g.ctx.Out

	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var data struct {
		Objects []*rubyObject
	}
	err = json.Unmarshal(out, &data)
	if err != nil {
		return nil, err
	}

	return data.Objects, nil
}
