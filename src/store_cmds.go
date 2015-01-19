package src

import (
	"fmt"
	"log"
	"path/filepath"

	"strings"

	"sourcegraph.com/sourcegraph/rwvfs"
	"sourcegraph.com/sourcegraph/srclib"
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
	lrepo, _ := openLocalRepo()
	if lrepo != nil && lrepo.RootDir != "" {
		relDir, err := filepath.Rel(absDir, lrepo.RootDir)
		if err == nil {
			SetOptionDefaultValue(c.Group, "root", filepath.Join(relDir, store.SrclibStoreDir))
		}
	}

	importC, err := c.AddCommand("import",
		"import data",
		`The import command imports data (from .srclib-cache) into the store.`,
		&storeImportCmd,
	)
	if err != nil {
		log.Fatal(err)
	}
	setDefaultRepoURIOpt(importC)
	setDefaultCommitIDOpt(importC)

	_, err = c.AddCommand("repos",
		"list repos",
		"The repos command lists all repos that match a filter.",
		&storeReposCmd,
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

type StoreCmd struct {
	Type string `short:"t" long:"type" description:"the (multi-)repo store type to use (RepoStore, MultiRepoStore, etc.)" default:"RepoStore"`
	Root string `short:"r" long:"root" description:"the root of the store (repo clone dir for RepoStore, global path for MultiRepoStore, etc.)"`
}

var storeCmd StoreCmd

func (c *StoreCmd) Execute(args []string) error { return nil }

// store returns the store specified by StoreCmd's Type and Root
// options.
func (c *StoreCmd) store() (interface{}, error) {
	conf := &store.FlatFileConfig{Codec: store.GobAndJSONGzipCodec{}}
	switch c.Type {
	case "RepoStore":
		return store.NewFlatFileRepoStore(rwvfs.OS(c.Root), conf), nil
	case "MultiRepoStore":
		return store.NewFlatFileMultiRepoStore(rwvfs.OS(c.Root), conf), nil
	default:
		return nil, fmt.Errorf("unrecognized store --type value: %q (valid values are RepoStore, MultiRepoStore)", c.Type)
	}
}

type StoreImportCmd struct {
	DryRun bool `short:"n" long:"dry-run" description:"print what would be done but don't do anything"`

	Repo     string `long:"repo" description:"only import for this repo"`
	Unit     string `long:"unit" description:"only import source units with this name"`
	UnitType string `long:"unit-type" description:"only import source units with this type"`
	CommitID string `long:"commit" description:"commit ID of commit whose data to import"`

	RemoteBuildData bool `long:"remote" description:"import remote build data (not the local .srclib-cache build data)"`
}

var storeImportCmd StoreImportCmd

func (c *StoreImportCmd) Execute(args []string) error {
	lrepo, err := openLocalRepo()
	if err != nil {
		return err
	}

	s, err := storeCmd.store()
	if err != nil {
		return err
	}

	bdfs, label, err := getBuildDataFS(!c.RemoteBuildData, c.Repo, c.CommitID)
	if err != nil {
		return err
	}
	if GlobalOpt.Verbose {
		log.Printf("# Importing build data for %s (commit %s) from %s", c.Repo, c.CommitID, label)
	}

	// Traverse the build data directory for this repo and commit to
	// create the makefile that lists the targets (which are the data
	// files we will import).
	treeConfig, err := config.ReadCached(bdfs)
	if err != nil {
		return err
	}
	mf, err := plan.CreateMakefile(".", nil, lrepo.VCSType, treeConfig, plan.Options{NoCache: true})
	if err != nil {
		return err
	}

	for _, rule := range mf.Rules {
		if c.Unit != "" || c.UnitType != "" {
			type ruleForSourceUnit interface {
				SourceUnit() *unit.SourceUnit
			}
			if rule, ok := rule.(ruleForSourceUnit); ok {
				u := rule.SourceUnit()
				if (c.Unit != "" && u.Name != c.Unit) || (c.UnitType != "" && u.Type != c.UnitType) {
					continue
				}
			} else {
				// Skip all non-source-unit rules if --unit or
				// --unit-type are specified.
				continue
			}
		}

		switch rule := rule.(type) {
		case *grapher.GraphUnitRule:
			var data graph.Output
			if err := readJSONFileFS(bdfs, rule.Target(), &data); err != nil {
				return err
			}
			if c.DryRun || GlobalOpt.Verbose {
				log.Printf("# Importing graph data (%d defs, %d refs, %d docs, %d anns) for unit %s %s", len(data.Defs), len(data.Refs), len(data.Docs), len(data.Anns), rule.Unit.Type, rule.Unit.Name)
				if c.DryRun {
					continue
				}
			}

			switch imp := s.(type) {
			case store.RepoImporter:
				if err := imp.Import(c.CommitID, rule.Unit, data); err != nil {
					return err
				}
			case store.MultiRepoImporter:
				if err := imp.Import(c.Repo, c.CommitID, rule.Unit, data); err != nil {
					return err
				}
			default:
				return fmt.Errorf("store (type %T) does not implement importing", s)
			}
		}
	}

	return nil
}

type StoreReposCmd struct {
	IDContains string `short:"i" long:"id-contains" description:"filter to repos whose ID contains this substring"`
}

var storeReposCmd StoreReposCmd

func (c *StoreReposCmd) Execute(args []string) error {
	s, err := storeCmd.store()
	if err != nil {
		return err
	}

	mrs, ok := s.(store.MultiRepoStore)
	if !ok {
		return fmt.Errorf("store (type %T) does not implement listing repositories", s)
	}

	repos, err := mrs.Repos(func(repo string) bool {
		v := true
		if c.IDContains != "" {
			v = v && strings.Contains(repo, c.IDContains)
		}
		return v
	})
	if err != nil {
		return err
	}
	for _, repo := range repos {
		fmt.Println(repo)
	}
	return nil
}

type StoreVersionsCmd struct {
	Repo           string `long:"repo"`
	CommitIDPrefix string `long:"commit" description:"commit ID prefix"`
}

var storeVersionsCmd StoreVersionsCmd

func (c *StoreVersionsCmd) Execute(args []string) error {
	s, err := storeCmd.store()
	if err != nil {
		return err
	}

	rs, ok := s.(store.RepoStore)
	if !ok {
		return fmt.Errorf("store (type %T) does not implement listing versions", s)
	}

	versions, err := rs.Versions(func(version *store.Version) bool {
		v := true
		if c.Repo != "" {
			v = v && c.Repo == version.Repo
		}
		if c.CommitIDPrefix != "" {
			v = v && strings.HasPrefix(version.CommitID, c.CommitIDPrefix)
		}
		return v
	})
	if err != nil {
		return err
	}
	for _, version := range versions {
		if version.Repo != "" {
			fmt.Print(version.Repo, "\t")
		}
		fmt.Println(version.CommitID)
	}
	return nil
}

type StoreUnitsCmd struct {
	Type     string `long:"type" `
	Name     string `long:"name"`
	CommitID string `long:"commit"`
	Repo     string `long:"repo"`

	File string `long:"file" description:"filter by units whose Files list contains this file"`
}

var storeUnitsCmd StoreUnitsCmd

func (c *StoreUnitsCmd) Execute(args []string) error {
	s, err := storeCmd.store()
	if err != nil {
		return err
	}

	ts, ok := s.(store.TreeStore)
	if !ok {
		return fmt.Errorf("store (type %T) does not implement listing source units", s)
	}

	units, err := ts.Units(func(unit *unit.SourceUnit) bool {
		v := true
		if c.Type != "" {
			v = v && c.Type == unit.Type
		}
		if c.Name != "" {
			v = v && c.Name == unit.Name
		}
		if c.Repo != "" {
			v = v && c.Repo == unit.Repo
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
	Repo     string `long:"repo"`
	Path     string `long:"path"`
	UnitType string `long:"unit-type" `
	Unit     string `long:"unit"`
	File     string `long:"file"`
	CommitID string `long:"commit"`

	NamePrefix string `long:"name-prefix"`
}

var storeDefsCmd StoreDefsCmd

func (c *StoreDefsCmd) Execute(args []string) error {
	s, err := storeCmd.store()
	if err != nil {
		return err
	}

	us, ok := s.(store.UnitStore)
	if !ok {
		return fmt.Errorf("store (type %T) does not implement listing defs", s)
	}

	defs, err := us.Defs(func(def *graph.Def) bool {
		v := true
		if c.Repo != "" {
			v = v && c.Repo == string(def.Repo)
		}
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

	Repo     string `long:"repo"`
	UnitType string `long:"unit-type" `
	Unit     string `long:"unit"`
	File     string `long:"file"`
	CommitID string `long:"commit"`
}

var storeRefsCmd StoreRefsCmd

func (c *StoreRefsCmd) Execute(args []string) error {
	s, err := storeCmd.store()
	if err != nil {
		return err
	}

	us, ok := s.(store.UnitStore)
	if !ok {
		return fmt.Errorf("store (type %T) does not implement listing refs", s)
	}

	refs, err := us.Refs(func(ref *graph.Ref) bool {
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
		if c.Repo != "" {
			v = v && c.Repo == string(ref.Repo)
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
