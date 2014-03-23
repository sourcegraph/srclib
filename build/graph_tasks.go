package build

import (
	"fmt"

	"sync"

	"sourcegraph.com/sourcegraph/graph"
	"sourcegraph.com/sourcegraph/srcgraph/grapher2"
	"sourcegraph.com/sourcegraph/srcgraph/task2"
)

func (p *repositoryPlanner) planGraphTasks() []task2.Task {
	var tasks []task2.Task

	outputs := make(map[string]*grapher2.Output)
	p.bd.Graph = outputs
	var outputsMu sync.Mutex
	var w sync.WaitGroup

	for _, u_ := range p.c.SourceUnits {
		u := u_
		w.Add(1)
		tasks = append(tasks, task2.NewTaskFunc(fmt.Sprintf("[%s] graph", u.ID()), p.x, func(x *task2.Context) error {
			defer w.Done()
			output, err := grapher2.Graph(p.dir, u, p.c, x)
			if err != nil {
				return err
			}
			outputsMu.Lock()
			defer outputsMu.Unlock()
			outputs[u.ID()] = output
			return nil
		}))
	}

	tasks = append(tasks, task2.NewTaskFunc("graph stats", p.x, func(x *task2.Context) error {
		w.Wait()
		xrefs := make(map[graph.SymbolKey]int)
		irefs := make(map[graph.SymbolKey]int)
		var nsyms, nrefs int

		for _, o := range outputs {
			for _, ref := range o.Refs {
				sk := ref.SymbolKey()
				if sk.Repo == p.c.URI { // iref
					irefs[sk]++
				} else { // xref
					xrefs[sk]++
				}
				nrefs++
			}
		}

		var symsWithIRefs int
		for _, o := range outputs {
			for _, sym := range o.Symbols {
				if irefs, present := irefs[sym.SymbolKey]; present {
					sym.IRefs = irefs
					if irefs > 0 {
						symsWithIRefs++
					}
				}
				nsyms++
			}
		}

		x.Log.Printf("Totals: %d symbols, %d refs", nsyms, nrefs)
		x.Log.Printf("Symbols with irefs: %d", symsWithIRefs)
		x.Log.Printf("External symbols with xrefs: %d", len(xrefs))
		if delta := len(irefs) - symsWithIRefs; delta > 0 {
			x.Log.Printf("Found %d distinct symbol keys with unresolved irefs:", delta)

			// Print out symbol keys with unresolved irefs.
			isyms := make(map[graph.SymbolKey]struct{})
			for _, o := range outputs {
				for _, isym := range o.Symbols {
					isyms[isym.SymbolKey] = struct{}{}
				}
			}
			for sk, irefs := range irefs {
				if _, present := isyms[sk]; !present {
					x.Log.Printf("\t%d irefs to nonexistent symbol %s", irefs, sk)
				}
			}
		}

		return nil
	}))

	return tasks
}
