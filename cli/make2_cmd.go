package cli

import (
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/alexsaveliev/go-colorable-wrapper"
	"sourcegraph.com/sourcegraph/go-flags"
	"sourcegraph.com/sourcegraph/srclib"
	"sourcegraph.com/sourcegraph/srclib/buildstore"
	"sourcegraph.com/sourcegraph/srclib/config"
	"sourcegraph.com/sourcegraph/srclib/plan"

	"sourcegraph.com/sourcegraph/makex"
)

func init() {
	cliInit = append(cliInit, func(cli *flags.Command) {
		_, err := cli.AddCommand("make2",
			"plans and executes plan",
			`Generates a plan (in Makefile form, in memory) for analyzing the tree and executes the plan. `,
			&makeCmd2,
		)
		if err != nil {
			log.Fatal(err)
		}
	})
}

type MakeCmd2 struct {
	Quiet  bool `short:"q" long:"quiet" description:"silence all output"`
	DryRun bool `short:"n" long:"dry-run" description:"print what would be done and exit"`

	Dir Directory `short:"C" long:"directory" description:"change to DIR before doing anything" value-name:"DIR"`

	Args struct {
		Goals []string `name:"GOALS..." description:"Makefile targets to build (default: all)"`
	} `positional-args:"yes"`
}

var makeCmd2 MakeCmd2

func (c *MakeCmd2) Execute(args []string) error {
	if c.Dir != "" {
		if err := os.Chdir(c.Dir.String()); err != nil {
			return err
		}
	}

	mf, err := CreateMakefile2()
	if err != nil {
		return err
	}

	goals := c.Args.Goals
	if len(goals) == 0 {
		if defaultRule := mf.DefaultRule(); defaultRule != nil {
			goals = []string{defaultRule.Target()}
		}
	}

	mkConf := &makex.Default
	mk := mkConf.NewMaker(mf, goals...)
	mk.Verbose = GlobalOpt.Verbose

	if c.Quiet {
		mk.RuleOutput = func(r makex.Rule) (out io.WriteCloser, err io.WriteCloser, logger *log.Logger) {
			return nopWriteCloser{}, nopWriteCloser{},
				log.New(nopWriteCloser{}, "", 0)
		}
	}

	if c.DryRun {
		return mk.DryRun(os.Stdout)
	}

	err = mk.Run()
	switch {
	case c.Quiet:
		// Skip output
	case err == nil:
		colorable.Println(colorable.Green("MAKE SUCCESS"))
	case err != nil:
		colorable.Println(colorable.DarkRed("MAKE FAILURE"))
	}
	return err
}

// CreateMakefile creates a Makefile to build a tree. The cwd should
// be the root of the tree you want to make (due to some probably
// unnecessary assumptions that CreateMaker makes).
func CreateMakefile2() (*makex.Makefile, error) {
	localRepo, err := OpenRepo(".")
	if err != nil {
		return nil, err
	}
	buildStore, err := buildstore.LocalRepo(localRepo.RootDir)
	if err != nil {
		return nil, err
	}

	config, err := config.ReadCached2(buildStore.Commit(localRepo.CommitID))
	if err != nil {
		return nil, err
	}
	if len(config.Units) == 0 {
		log.Printf("No source unit files found. Did you mean to run `%s config`? (This is not an error; it just means that srclib didn't find anything to build or analyze here.)", srclib.CommandName)
	}

	// TODO(sqs): buildDataDir is hardcoded.
	buildDataDir := filepath.Join(buildstore.BuildDataDirName, localRepo.CommitID)
	mf, err := plan.CreateMakefile2(buildDataDir, buildStore, localRepo.VCSType, config)
	if err != nil {
		return nil, err
	}
	return mf, nil
}
