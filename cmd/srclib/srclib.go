package main

import (
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"strings"

	"sourcegraph.com/sourcegraph/srclib/cli"
	_ "sourcegraph.com/sourcegraph/srclib/dep"
	_ "sourcegraph.com/sourcegraph/srclib/scan"
)

func main() {
	if cpuprof := os.Getenv("CPUPROF"); cpuprof != "" {
		f, err := os.Create(cpuprof)
		if err != nil {
			log.Fatal("CPUPROF:", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("StartCPUProfile:", err)
		}
		defer func() {
			pprof.StopCPUProfile()
			f.Close()
		}()
	}

	if err := cli.Main(); err != nil {
		fmt.Fprintf(os.Stderr, "FAILED: %s - %s\n", strings.Join(os.Args, " "), err.Error())
		os.Exit(1)
	}
}
