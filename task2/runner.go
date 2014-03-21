package task2

import (
	"log"
	"sync"
)

func Run(tasks []Task) *sync.WaitGroup {
	var w sync.WaitGroup
	for _, t_ := range tasks {
		t := t_
		w.Add(1)
		t.Start()
		go func() {
			defer w.Done()
			err := t.Wait()
			if err != nil {
				log.Printf("Task %q failed: %s.", t.Name(), err)
			}
		}()
	}
	return &w
}
