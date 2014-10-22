package src

import (
	"encoding/json"
	"log"
	"os"

	"github.com/sqs/go-flags"

	"sourcegraph.com/sourcegraph/srclib/grapher"
	"sourcegraph.com/sourcegraph/srclib/unit"
	"sourcegraph.com/sourcegraph/srclib/vcsutil"
)

func init() {
	c, err := CLI.AddCommand("internal", "(internal subcommands - do not use)", "Internal subcommands. Do not use.", &struct{}{})
	if err != nil {
		log.Fatal(err)
	}

	_, err = c.AddCommand("normalize-graph-data", "", "", &normalizeGraphDataCmd)
	if err != nil {
		log.Fatal(err)
	}

	_, err = c.AddCommand("unit-blame", "", "", &unitBlameCmd)
	if err != nil {
		log.Fatal(err)
	}
}

type NormalizeGraphDataCmd struct {
	UnitType string `long:"unit-type" description:"source unit type (e.g., GoPackage)"`
	Dir      string `long:"dir" description:"directory of source unit (SourceUnit.Dir field)"`
}

var normalizeGraphDataCmd NormalizeGraphDataCmd

func (c *NormalizeGraphDataCmd) Execute(args []string) error {
	in := os.Stdin

	var o *grapher.Output
	if err := json.NewDecoder(in).Decode(&o); err != nil {
		return err
	}

	if err := grapher.NormalizeData(c.UnitType, c.Dir, o); err != nil {
		return err
	}

	data, err := json.MarshalIndent(o, "", "  ")
	if err != nil {
		return err
	}

	if _, err := os.Stdout.Write(data); err != nil {
		return err
	}

	return nil
}

type UnitBlameCmd struct {
	UnitData flags.Filename `long:"unit-data" required:"yes" description:"source unit definition JSON file" value-name:"FILE"`
}

var unitBlameCmd UnitBlameCmd

func (c *UnitBlameCmd) Execute(args []string) error {
	var u *unit.SourceUnit
	if err := readJSONFile(string(c.UnitData), &u); err != nil {
		return err
	}

	currentRepo, err := OpenRepo(".")
	if err != nil {
		return err
	}

	paths, err := unit.ExpandPaths(currentRepo.RootDir, u.Files)
	if err != nil {
		log.Fatal(err)
	}

	var out0 *vcsutil.BlameOutput
	if paths == nil {
		out0, err = vcsutil.BlameRepository(currentRepo.RootDir, currentRepo.CommitID)
	} else {
		out0, err = vcsutil.BlameFiles(currentRepo.RootDir, paths, currentRepo.CommitID)
	}
	if err != nil {
		log.Fatal(err)
	}

	out, err := json.MarshalIndent(out0, "", "  ")
	if err != nil {
		return err
	}

	if _, err := os.Stdout.Write(out); err != nil {
		return err
	}

	return nil
}
