package build

import (
	"path/filepath"
	"strconv"

	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
	"sourcegraph.com/sourcegraph/srcgraph/util2/makefile"
)

func init() {
	RegisterRuleMaker("dep", makeDepRules)
}

func makeDepRules(c *config.Repository, commitID string) ([]makefile.Rule, error) {
	if len(c.SourceUnits) == 0 {
		return nil, nil
	}

	resolveRule := &ResolveDepsRule{
		RepositoryCommitSpec: RepositoryCommitSpec{
			RepositoryURI: c.URI,
			CommitID:      commitID,
		},
	}

	rules := []makefile.Rule{resolveRule}
	for _, u := range c.SourceUnits {
		rule := &ListSourceUnitDepsRule{
			SourceUnitSpec{
				RepositoryURI: c.URI,
				CommitID:      commitID,
				Unit:          u,
			},
		}
		rules = append(rules, rule)
		resolveRule.RawDepLists = append(resolveRule.RawDepLists, rule.Target())
	}

	return rules, nil
}

type ResolveDepsRule struct {
	RepositoryCommitSpec
	RawDepLists []makefile.Target
}

func (r *ResolveDepsRule) Target() makefile.Target {
	return &RepositoryCommitOutputFile{r.RepositoryCommitSpec, "resolved_deps"}
}

func (r *ResolveDepsRule) Prereqs() []string {
	var files []string
	for _, rawDepListFile := range r.RawDepLists {
		files = append(files, rawDepListFile.Name())
	}
	return files
}

func (r *ResolveDepsRule) Recipes() []makefile.Recipe {
	return []makefile.Recipe{
		makefile.CommandRecipe{"mkdir", "-p", strconv.Quote(filepath.Dir(r.Target().Name()))},
		makefile.CommandRecipe{"srcgraph", "-v", "resolve-deps", "-json", "$^", "1> $@"},
	}
}

type ListSourceUnitDepsRule struct {
	SourceUnitSpec
}

func (r *ListSourceUnitDepsRule) Target() makefile.Target {
	return &SourceUnitOutputFile{r.SourceUnitSpec, "raw_deps"}
}

func (r *ListSourceUnitDepsRule) Prereqs() []string { return r.Unit.Paths() }

func (r *ListSourceUnitDepsRule) Recipes() []makefile.Recipe {
	return []makefile.Recipe{
		makefile.CommandRecipe{"mkdir", "-p", strconv.Quote(filepath.Dir(r.Target().Name()))},
		makefile.CommandRecipe{"srcgraph", "-v", "list-deps", "-json", strconv.Quote(string(unit.MakeID(r.Unit))), "1> $@"},
	}
}

// func (p *repositoryPlanner) planDepTasks() []task2.Task {
// 	rawDepChan := make(chan *dep2.RawDependency, 100)
// 	var activeListers sync.WaitGroup

// 	var tasks []task2.Task

// 	for _, u_ := range p.c.SourceUnits {
// 		u := u_
// 		activeListers.Add(1)
// 		tasks = append(tasks, task2.NewTaskFunc(fmt.Sprintf("[%s] raw deps", u.ID()), p.x, func(x *task2.Context) error {
// 			defer activeListers.Done()
// 			rawDeps, err := dep2.List(p.dir, u, p.c, x)
// 			if err != nil {
// 				return err
// 			}
// 			for _, rawDep := range rawDeps {
// 				rawDepChan <- rawDep
// 			}
// 			return nil
// 		}))
// 	}

// 	go func() {
// 		activeListers.Wait()
// 		close(rawDepChan)
// 	}()

// 	tasks = append(tasks, &resolveDepsTask{
// 		rawDep: rawDepChan,
// 		c:      p.c,
// 		x:      p.x.Child(),
// 	})

// 	return tasks
// }

// type resolveDepsTask struct {
// 	rawDep    <-chan *dep2.RawDependency
// 	resolving *dep2.RawDependency
// 	resolved  []*dep2.ResolvedTarget

// 	resolveCache map[*dep2.RawDependency]*dep2.ResolvedTarget

// 	errs util2.Errors
// 	c    *config.Repository
// 	x    *task2.Context

// 	started, done bool
// 	doneChan      chan struct{}
// }

// func (t *resolveDepsTask) Name() string { return "resolve deps" }

// func (t *resolveDepsTask) Context() *task2.Context { return t.x }

// func (t *resolveDepsTask) Start() {
// 	if t.started {
// 		panic("resolveDepsTask: already started")
// 	}
// 	t.started = true
// 	t.doneChan = make(chan struct{})
// 	t.resolveCache = make(map[*dep2.RawDependency]*dep2.ResolvedTarget)
// 	go func() {
// 		for rawDep := range t.rawDep {
// 			t.resolving = rawDep
// 			var resolvedDep *dep2.ResolvedTarget

// 			// look up in cache
// 			for bd, rt := range t.resolveCache {
// 				if rawDep.TargetType == bd.TargetType && rawDep.Target == bd.Target {
// 					resolvedDep = rt
// 					break
// 				}
// 			}

// 			if resolvedDep == nil {
// 				// not found in cache
// 				resolvedDep, err := dep2.Resolve(rawDep, t.c, t.x)
// 				if err != nil {
// 					t.errs = append(t.errs, err)
// 					t.resolving = nil
// 					continue

// 				}
// 				t.resolved = append(t.resolved, resolvedDep)
// 			}

// 			t.resolveCache[rawDep] = resolvedDep
// 			t.resolving = nil
// 		}
// 		close(t.doneChan)
// 	}()
// }

// func (t *resolveDepsTask) Wait() error {
// 	<-t.doneChan
// 	if len(t.errs) == 0 {
// 		return nil
// 	}
// 	return t.errs
// }

// func (t *resolveDepsTask) Status() string {
// 	if t.done {
// 		return "done"
// 	} else if t.resolving != nil {
// 		return fmt.Sprintf("resolving %s %v", t.resolving.TargetType, t.resolving.Target)
// 	} else if t.started {
// 		return "waiting"
// 	}
// 	return "pending"
// }

// func (t *resolveDepsTask) Summary() string {
// 	if t.started {
// 		return fmt.Sprintf("(%d queued, %d resolved, %d errors)", len(t.rawDep), len(t.resolved), len(t.errs))
// 	}
// 	return ""
// }
