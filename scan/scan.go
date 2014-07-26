package scan

import (
	"runtime"
	"sync"

	"code.google.com/p/rog-go/parallel"
	"github.com/sourcegraph/srclib/config"
	"github.com/sourcegraph/srclib/toolchain"
	"github.com/sourcegraph/srclib/unit"
)

type Command struct {
	config.Command
}

func ScanMulti(scanners []toolchain.Tool, cmd Command) ([]*unit.SourceUnit, error) {
	var (
		units []*unit.SourceUnit
		mu    sync.Mutex
	)

	run := parallel.NewRun(runtime.GOMAXPROCS(0))
	for _, scanner_ := range scanners {
		scanner := scanner_
		run.Do(func() error {
			units2, err := Scan(scanner, cmd)
			if err != nil {
				return err
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

func Scan(scanner toolchain.Tool, cmd Command) ([]*unit.SourceUnit, error) {
	args, err := toolchain.MarshalArgs(&cmd)
	if err != nil {
		return nil, err
	}

	var units []*unit.SourceUnit
	if err := scanner.Run(args, &units); err != nil {
		return nil, err
	}

	return units, nil
}
