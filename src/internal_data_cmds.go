package src

import (
	"encoding/json"
	"log"
	"os"

	"github.com/sourcegraph/srclib/grapher2"
)

func init() {
	c, err := CLI.AddCommand("internal", "(internal subcommands - do not use)", "", &struct{}{})
	if err != nil {
		log.Fatal(err)
	}

	_, err = c.AddCommand("normalize-graph-data", "", "", &normalizeGraphDataCmd)
	if err != nil {
		log.Fatal(err)
	}
}

type NormalizeGraphDataCmd struct{}

var normalizeGraphDataCmd NormalizeGraphDataCmd

func (c *NormalizeGraphDataCmd) Execute(args []string) error {
	in := os.Stdin

	var o *grapher2.Output
	if err := json.NewDecoder(in).Decode(&o); err != nil {
		return err
	}

	if err := grapher2.NormalizeData(o); err != nil {
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
