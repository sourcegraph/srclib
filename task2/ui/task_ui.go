package ui

import (
	"fmt"
	"log"
	"time"

	tm "github.com/buger/goterm"
	"sourcegraph.com/sourcegraph/srcgraph/task2"
)

func Start(bufferedUI bool, tasks []task2.Task, x *task2.Context) {
	go func() {
		if bufferedUI {
			for {
				tm.Clear()
				tm.MoveCursor(1, 1)

				for _, t := range tasks {
					tm.Printf("%s %s\n", tm.Color(t.Name(), tm.CYAN), t.Summary())
					tm.Printf("\t%s\n", t.Status())
					tm.Println(tm.Color(fmt.Sprintf("logs: %s", t.Context().Destination), tm.BLACK))
					tm.Println()
				}
				tm.Println()
				tm.Println()
				tm.Printf("Logging to: %s\n", x.Destination)

				tm.Flush()
				time.Sleep(time.Millisecond * 500)
			}
		} else {
			for {
				log.Printf("======= TASK STATUS =======")
				for _, t := range tasks {
					log.Printf("%s: %s", t.Name(), t.Status())
				}
				log.Printf("===========================")
				log.Println()
				time.Sleep(time.Second)
			}
		}
	}()
}

func List(tasks []task2.Task) {
	fmt.Println("======= TASKS =======")
	for _, t := range tasks {
		fmt.Println(t.Name())
	}
}
