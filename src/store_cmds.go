package src

import (
	"fmt"
	"log"
	"path/filepath"

	"strings"

	"sourcegraph.com/sourcegraph/rwvfs"
	"sourcegraph.com/sourcegraph/srclib"
	"sourcegraph.com/sourcegraph/srclib/buildstore"
	"sourcegraph.com/sourcegraph/srclib/config"
	"sourcegraph.com/sourcegraph/srclib/graph"
	"sourcegraph.com/sourcegraph/srclib/grapher"
	"sourcegraph.com/sourcegraph/srclib/graphstore"
	"sourcegraph.com/sourcegraph/srclib/plan"
	"sourcegraph.com/sourcegraph/srclib/store"
	"sourcegraph.com/sourcegraph/srclib/unit"
)

func init() {
	c, err := CLI.AddCommand("store",
		"graph store commands",
		"",
		&storeCmd,
	)
	if err != nil {
		log.Fatal(err)
	}

	_, err = c.AddCommand("import",
		"import data",
		`The import command imports data (from .srclib-cache) into the store.`,
		&storeImportCmd,
	)
	if err != nil {
		log.Fatal(err)
	}

	_, err = c.AddCommand("versions",
		"list versions",
		"The versions command lists all versions that match a filter.",
		&storeVersionsCmd,
	)
	if err != nil {
		log.Fatal(err)
	}

	_, err = c.AddCommand("units",
		"list units",
		"The units command lists all units that match a filter.",
		&storeUnitsCmd,
	)
	if err != nil {
		log.Fatal(err)
	}

	_, err = c.AddCommand("defs",
		"list defs",
		"The defs command lists all defs that match a filter.",
		&storeDefsCmd,
	)
	if err != nil {
		log.Fatal(err)
	}

	_, err = c.AddCommand("refs",
		"list refs",
		"The refs command lists all refs that match a filter.",
		&storeRefsCmd,
	)
	if err != nil {
		log.Fatal(err)
	}
}

var graphStore *graphstore.Store

func init() {
	var err error
	graphStore, err = graphstore.NewLocal(filepath.Join(srclib.Path, graphstore.Name))
	if err != nil {
		log.Fatal(err)
	}
}

type StoreCmd struct{}

var storeCmd StoreCmd

func (c *StoreCmd) Execute(args []string) error { return nil }

type StoreImportCmd struct {
	DryRun bool `short:"n" long:"dry-run" description:"print what would be done but don't do anything"`
}

var storeImportCmd StoreImportCmd

func (c *StoreImportCmd) Execute(args []string) error {
	lrepo, err := openLocalRepo()
	if err != nil {
		return err
	}

	rs, err := openRepoStore()
	if err != nil {
		return err
	}

	// Open the build data cache.
	buildStore, err := buildstore.LocalRepo(lrepo.RootDir)
	if err != nil {
		return err
	}
	bdfs := buildStore.Commit(lrepo.CommitID)

	// Traverse the build data directory for this repo and commit to
	// create the makefile that lists the targets (which are the data
	// files we will import).
	treeConfig, err := config.ReadCached(bdfs)
	if err != nil {
		return err
	}
	mf, err := plan.CreateMakefile(".", buildStore, lrepo.VCSType, treeConfig, plan.Options{})
	if err != nil {
		return err
	}
	for _, rule := range mf.Rules {
		switch rule := rule.(type) {
		case *grapher.GraphUnitRule:
			var data graph.Output
			if err := readJSONFileFS(bdfs, rule.Target(), &data); err != nil {
				return err
			}
			if c.DryRun || GlobalOpt.Verbose {
				log.Printf("# Import graph data (%d defs, %d refs, %d docs, %d anns) for commit %s unit %s %s", len(data.Defs), len(data.Refs), len(data.Docs), len(data.Anns), lrepo.CommitID, rule.Unit.Type, rule.Unit.Name)
				if c.DryRun {
					continue
				}
			}
			if err := rs.Import(lrepo.CommitID, rule.Unit, data); err != nil {
				return err
			}
		}
	}

	return nil
}

type StoreVersionsCmd struct{}

var storeVersionsCmd StoreVersionsCmd

func (c *StoreVersionsCmd) Execute(args []string) error {
	rs, err := openRepoStore()
	if err != nil {
		return err
	}

	versions, err := rs.Versions(nil)
	if err != nil {
		return err
	}
	for _, version := range versions {
		fmt.Println(version.CommitID)
	}
	return nil
}

type StoreUnitsCmd struct {
	Type     string `long:"type" `
	Name     string `long:"name"`
	CommitID string `long:"commit"`

	File string `long:"file" description:"filter by units whose Files list contains this file"`
}

var storeUnitsCmd StoreUnitsCmd

func (c *StoreUnitsCmd) Execute(args []string) error {
	rs, err := openRepoStore()
	if err != nil {
		return err
	}

	units, err := rs.Units(func(unit *unit.SourceUnit) bool {
		v := true
		if c.Type != "" {
			v = v && c.Type == unit.Type
		}
		if c.Name != "" {
			v = v && c.Name == unit.Name
		}
		if c.CommitID != "" {
			v = v && c.CommitID == unit.CommitID
		}
		if c.File != "" {
			hasFile := false
			c.File = filepath.Clean(c.File)
			for _, file := range unit.Files {
				if filepath.Clean(file) == c.File {
					hasFile = true
					break
				}
			}
			v = v && hasFile
		}
		return v
	})
	if err != nil {
		return err
	}
	PrintJSON(units, "  ")
	return nil
}

type StoreDefsCmd struct {
	Path     string `long:"path"`
	UnitType string `long:"unit-type" `
	Unit     string `long:"unit"`
	File     string `long:"file"`
	CommitID string `long:"commit"`

	NamePrefix string `long:"name-prefix"`
}

var storeDefsCmd StoreDefsCmd

func (c *StoreDefsCmd) Execute(args []string) error {
	rs, err := openRepoStore()
	if err != nil {
		return err
	}

	defs, err := rs.Defs(func(def *graph.Def) bool {
		v := true
		if c.Path != "" {
			v = v && c.Path == string(def.Path)
		}
		if c.Unit != "" {
			v = v && c.Unit == def.Unit
		}
		if c.File != "" {
			v = v && c.File == def.File
		}
		if c.CommitID != "" {
			v = v && c.CommitID == def.CommitID
		}
		if c.NamePrefix != "" {
			v = v && strings.HasPrefix(def.Name, c.NamePrefix)
		}
		return v
	})
	if err != nil {
		return err
	}
	PrintJSON(defs, "  ")
	return nil
}

type StoreRefsCmd struct {
	DefRepo     string `long:"def-repo"`
	DefUnitType string `long:"def-unit-type" `
	DefUnit     string `long:"def-unit"`
	DefPath     string `long:"def-path"`

	UnitType string `long:"unit-type" `
	Unit     string `long:"unit"`
	File     string `long:"file"`
	CommitID string `long:"commit"`
}

var storeRefsCmd StoreRefsCmd

func (c *StoreRefsCmd) Execute(args []string) error {
	rs, err := openRepoStore()
	if err != nil {
		return err
	}

	refs, err := rs.Refs(func(ref *graph.Ref) bool {
		v := true
		if c.DefRepo != "" {
			v = v && c.DefRepo == ref.DefRepo
		}
		if c.DefUnitType != "" {
			v = v && c.DefUnitType == ref.DefUnitType
		}
		if c.DefUnit != "" {
			v = v && c.DefUnit == ref.DefUnit
		}
		if c.DefPath != "" {
			v = v && c.DefPath == ref.DefPath
		}
		if c.UnitType != "" {
			v = v && c.UnitType == ref.UnitType
		}
		if c.Unit != "" {
			v = v && c.Unit == ref.Unit
		}
		if c.File != "" {
			v = v && c.File == ref.File
		}
		if c.CommitID != "" {
			v = v && c.CommitID == ref.CommitID
		}
		return v
	})
	if err != nil {
		return err
	}
	PrintJSON(refs, "  ")
	return nil
}

func openRepoStore() (store.RepoStoreImporter, error) {
	lrepo, err := openLocalRepo()
	if err != nil {
		return nil, err
	}
	conf := &store.FlatFileConfig{Codec: store.GobAndJSONGzipCodec{}}
	rs := store.NewFlatFileRepoStore(rwvfs.OS(filepath.Join(lrepo.RootDir, ".srclib-store")), conf)
	return rs, nil
}
