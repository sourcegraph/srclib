package build

import (
	"sourcegraph.com/sourcegraph/srcgraph/task2"
	"sourcegraph.com/sourcegraph/vcs"
)

func (p *repositoryPlanner) planVCSTasks() []task2.Task {
	var tasks []task2.Task

	tasks = append(tasks, task2.NewTaskFunc("VCS metadata", p.x, func(x *task2.Context) error {
		_, _, err := vcs.Blame(p.dir, p.commitID, p.c, x)
		if err != nil {
			return err
		}
		return nil
	}))

	return tasks
}
