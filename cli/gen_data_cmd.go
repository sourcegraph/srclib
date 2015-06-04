package cli

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"sourcegraph.com/sourcegraph/srclib/dep"
	"sourcegraph.com/sourcegraph/srclib/graph"
	"sourcegraph.com/sourcegraph/srclib/unit"
)

func init() {
	c, err := CLI.AddCommand("gen-data",
		"generates fake data",
		`generates fake data and outputs to .srclib-cache for debugging imports.`,
		&genDataCmd,
	)
	if err != nil {
		log.Fatal(err)
	}
	c.Aliases = []string{"c"}
}

type GenDataCmd struct {
	Repo     string `short:"r" long:"repo" description:"repo to build" required:"yes"`
	CommitID string `short:"c" long:"commit" description:"commit ID to build" required:"yes"`
	NUnits   int    `long:"nunits" description:"number of units to generate" default:"1"`
	NFiles   int    `long:"nfiles" description:"number of files to generate per unit" default:"1"`
	NDefs    int    `long:"ndefs" description:"number of defs to generate per file" required:"yes"`
	NRefs    int    `long:"nrefs" description:"number of refs to generate per file" required:"yes"`

	GenSource bool `long:"gen-source" description:"whether to emit source files for the generated data"`
}

var genDataCmd GenDataCmd

func (c *GenDataCmd) Execute(args []string) error {
	for u := 0; u < c.NUnits; u++ {
		ut := &unit.SourceUnit{
			Name:     fmt.Sprintf("unit/%d", u),
			Type:     "GoPackage",
			Repo:     c.Repo,
			CommitID: c.CommitID,
			Files:    []string{},
			Dir:      fmt.Sprintf("unit/%d", u),
		}

		defs := make([]*graph.Def, c.NDefs)
		refs := make([]*graph.Ref, c.NRefs)
		// docs := make([]*graph.Doc, c.NDefs)
		docs := make([]*graph.Doc, 0)

		for f := 0; f < c.NFiles; f++ {
			filename := filepath.Join(fmt.Sprintf("unit_%d", u), "subpackage", fmt.Sprintf("file%d.go", f))
			offset := 0
			defName := "foo"

			ut.Files = append(ut.Files, filename)

			var sourceFile *os.File
			if c.GenSource {
				err := os.MkdirAll(filepath.Dir(filename), 0700)
				if err != nil {
					return err
				}
				file, err := os.Create(filename)
				if err != nil {
					return err
				}
				sourceFile = file
			}

			for i := 0; i < c.NDefs; i++ {
				defs[i] = &graph.Def{
					DefKey: graph.DefKey{
						Repo:     ut.Repo,
						CommitID: ut.CommitID,
						UnitType: ut.Type,
						Unit:     ut.Name,
						Path:     filepath.Join("package", "subpackage", "type", fmt.Sprintf("method_%d_%d", f, i)),
					},
					Name:     defName,
					Exported: true,
					File:     filename,
					DefStart: uint32(offset),
					DefEnd:   uint32(offset + len(defName)),
				}
				// docs[i] = &graph.Doc{
				// 	DefKey: defs[i].DefKey,
				// 	Data:   "I am a dostring",
				// 	File:   defs[i].File,
				// 	Start:  42,
				// 	End:    203,
				// }

				if sourceFile != nil {
					_, err := sourceFile.WriteString(defs[i].Name + "\n")
					if err != nil {
						return err
					}
				}

				offset += len(defName) + 1
			}

			for i, defIdx := 0, 0; i < c.NRefs; i, defIdx = i+1, (defIdx+1)%c.NDefs {
				refs[i] = &graph.Ref{
					DefRepo:     ut.Repo,
					DefUnitType: ut.Type,
					DefUnit:     ut.Name,
					DefPath:     filepath.Join("package", "subpackage", "type", fmt.Sprintf("method_%d_%d", f, defIdx)),
					Repo:        ut.Repo,
					CommitID:    ut.CommitID,
					UnitType:    ut.Type,
					Unit:        ut.Name,
					Def:         false,
					File:        filename,
					Start:       uint32(offset),
					End:         uint32(offset + len(defName)),
				}

				if sourceFile != nil {
					_, err := sourceFile.WriteString(defName + "\n")
					if err != nil {
						return err
					}
				}

				offset += len(defName) + 1
			}

			// Close source file
			if sourceFile != nil {
				sourceFile.Close()
			}
		}

		gr := graph.Output{Defs: defs, Refs: refs, Docs: docs}

		dp := make([]*dep.Resolution, 0)

		unitDir := filepath.Join(".srclib-cache", ut.CommitID, ut.Name)
		if err := os.MkdirAll(unitDir, 0700); err != nil {
			return err
		}

		unitFile, err := os.OpenFile(filepath.Join(unitDir, fmt.Sprintf("%s.unit.json", ut.Type)), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0666)
		if err != nil {
			return err
		}
		defer unitFile.Close()

		if err := json.NewEncoder(unitFile).Encode(ut); err != nil {
			return err
		}

		graphFile, err := os.OpenFile(filepath.Join(unitDir, fmt.Sprintf("%s.graph.json", ut.Type)), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0666)
		if err != nil {
			return err
		}
		defer graphFile.Close()

		if err := json.NewEncoder(graphFile).Encode(gr); err != nil {
			return err
		}

		depFile, err := os.OpenFile(filepath.Join(unitDir, fmt.Sprintf("%s.depresolve.json", ut.Type)), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0666)
		if err != nil {
			return err
		}
		defer depFile.Close()

		if err := json.NewEncoder(depFile).Encode(dp); err != nil {
			return err
		}
	}

	return nil
}
