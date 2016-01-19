package scan

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"sync"

	"github.com/rogpeppe/rog-go/parallel"
	"sourcegraph.com/sourcegraph/srclib/flagutil"
	"sourcegraph.com/sourcegraph/srclib/unit"
)

type Options struct {
	// Quiet silences all output.
	Quiet bool
}

// ScanMulti runs multiple scanner tools in parallel. It passes command-line
// options from opt to each one, and it sends the JSON representation of cfg
// (the repo/tree's Config) to each tool's stdin.
func ScanMulti(scanners [][]string, opt Options, treeConfig map[string]interface{}) ([]*unit.SourceUnit, error) {
	if treeConfig == nil {
		treeConfig = map[string]interface{}{}
	}

	var (
		units []*unit.SourceUnit
		mu    sync.Mutex
	)

	run := parallel.NewRun(runtime.GOMAXPROCS(0))
	for _, scanner_ := range scanners {
		scanner := scanner_
		run.Do(func() error {
			units2, err := Scan(scanner, opt, treeConfig)
			if err != nil {
				return fmt.Errorf("scanner %v: %s", scanner, err)
			}

			mu.Lock()
			defer mu.Unlock()
			units = append(units, units2...)
			return nil
		})
	}
	err := run.Wait()
	// Return error only if none of the commands succeeded.
	if len(units) == 0 {
		return nil, err
	}
	return units, nil
}

func Scan(scanner []string, opt Options, treeConfig map[string]interface{}) ([]*unit.SourceUnit, error) {
	args, err := flagutil.MarshalArgs(&opt)
	if err != nil {
		return nil, err
	}

	var errw bytes.Buffer
	cmd := exec.Command(scanner[0], scanner[1])
	cmd.Args = append(cmd.Args, args...)
	if opt.Quiet {
		cmd.Stderr = &errw
	} else {
		cmd.Stderr = io.MultiWriter(&errw, os.Stderr)
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	defer func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}()

	// Write the treeConfig into stdin.
	w := bufio.NewWriter(stdin)
	if err := json.NewEncoder(w).Encode(treeConfig); err != nil {
		w.Flush()
		return nil, err
	}
	if err := w.Flush(); err != nil {
		return nil, err
	}
	if err := stdin.Close(); err != nil {
		return nil, err
	}

	// Read on stdout into the list of source units.
	var units []*unit.SourceUnit
	if err := json.NewDecoder(stdout).Decode(&units); err != nil {
		return nil, err
	}
	if err := cmd.Wait(); err != nil {
		return nil, err
	}

	return units, nil
}
