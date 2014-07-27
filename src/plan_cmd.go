package src

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/sourcegraph/makex"
	"github.com/sourcegraph/srclib/buildstore"
	"github.com/sourcegraph/srclib/config"
	"github.com/sourcegraph/srclib/plan"
	"github.com/sqs/go-flags"
)

func init() {
	c, err := CLI.AddCommand("plan",
		"generate a Makefile to process a project",
		`Generate a Makefile to process a repository or directory tree.

If CONFIG-FILE is "-", it is read from stdin. If no CONFIG-FILE is given, "src config --output json" is executed in the current directory and its output is used as the configuration.
`,
		&planCmd,
	)
	if err != nil {
		log.Fatal(err)
	}

	_ = c
}

type PlanCmd struct {
	Args struct {
		ConfigFile flags.Filename `name:"CONFIG-FILE" description:"project config JSON file produced by 'src config' (unset means stdin)"`
	} `positional-args:"yes"`
}

var planCmd PlanCmd

func (c *PlanCmd) Execute(args []string) error {
	var input io.ReadCloser
	if c.Args.ConfigFile == "" {
		// If no config, then run config ourselves with the default options.
		configCmd := exec.Command("src", "config", "--output", "json", "--no-cache")
		configCmd.Stderr = os.Stderr
		out, err := configCmd.Output()
		if err != nil {
			return err
		}
		input = ioutil.NopCloser(bytes.NewReader(out))
	} else if c.Args.ConfigFile == "-" {
		input = os.Stdin
	} else {
		f, err := os.Open(string(c.Args.ConfigFile))
		if err != nil {
			return err
		}
		input = f
	}
	var cfg *config.Repository
	if err := json.NewDecoder(input).Decode(&cfg); err != nil {
		input.Close()
		return err
	}
	input.Close()

	currentRepo, err := OpenRepo(Dir)
	if err != nil {
		return err
	}
	buildStore, err := buildstore.NewRepositoryStore(currentRepo.RootDir)
	if err != nil {
		return err
	}
	buildDataDir, err := buildstore.BuildDir(buildStore, currentRepo.CommitID)
	if err != nil {
		return err
	}
	buildDataDir, _ = filepath.Rel(absDir, buildDataDir)

	mf, err := plan.CreateMakefile(buildDataDir, cfg)
	if err != nil {
		return err
	}

	mfData, err := makex.Marshal(mf)
	if err != nil {
		log.Fatal(err)
	}
	os.Stdout.Write(mfData)

	return nil
}
