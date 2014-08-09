package scan

import (
	"fmt"
	"runtime"
	"sync"

	"code.google.com/p/rog-go/parallel"
	"sourcegraph.com/sourcegraph/srclib/config"
	"sourcegraph.com/sourcegraph/srclib/repo"
	"sourcegraph.com/sourcegraph/srclib/toolchain"
	"sourcegraph.com/sourcegraph/srclib/unit"
)

type Options struct {
	config.Options
}

func ScanMulti(scanners []toolchain.Tool, opt Options) ([]*unit.SourceUnit, error) {
	var (
		units []*unit.SourceUnit
		mu    sync.Mutex
	)

	run := parallel.NewRun(runtime.GOMAXPROCS(0))
	for _, scanner_ := range scanners {
		scanner := scanner_
		run.Do(func() error {
			units2, err := Scan(scanner, opt)
			if err != nil {
				cmd, _ := scanner.Command()
				return fmt.Errorf("scanner %v: %s", cmd.Args, err)
			}

			mu.Lock()
			defer mu.Unlock()
			units = append(units, units2...)
			return nil
		})
	}
	if err := run.Wait(); err != nil {
		return nil, err
	}
	return units, nil
}

func Scan(scanner toolchain.Tool, opt Options) ([]*unit.SourceUnit, error) {
	args, err := toolchain.MarshalArgs(&opt)
	if err != nil {
		return nil, err
	}

	var units []*unit.SourceUnit
	if err := scanner.Run(args, nil, &units); err != nil {
		return nil, err
	}

	for _, u := range units {
		u.Repo = repo.URI(opt.Repo)
	}

	return units, nil
}
