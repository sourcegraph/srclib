package build

import (
	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/task2"
)

type repositoryPlanner struct {
	dir      string
	commitID string
	x        *task2.Context
	c        *config.Repository

	rd *RepositoryData
}

func (p *repositoryPlanner) planTasks() ([]task2.Task, *RepositoryData, error) {
	var tasks []task2.Task

	p.rd = &RepositoryData{
		Config:   p.c,
		CommitID: p.commitID,
	}

	tasks = append(tasks, p.planVCSTasks()...)
	tasks = append(tasks, p.planDepTasks()...)
	tasks = append(tasks, p.planGraphTasks()...)

	return tasks, p.rd, nil
}
