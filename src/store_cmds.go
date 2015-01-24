package src

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"path"
	"path/filepath"
	"strings"

	"sourcegraph.com/sourcegraph/rwvfs"
	"sourcegraph.com/sourcegraph/s3vfs"
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

	defsC, err := c.AddCommand("defs",
		"list defs",
		"The defs command lists all defs that match a filter.",
		&storeDefsCmd,
	)
	if err != nil {
		log.Fatal(err)
	}
	defsC.Aliases = []string{"def"}

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
	Type   string `short:"t" long:"type" description:"the (multi-)repo store type to use (RepoStore, MultiRepoStore, etc.)" default:"RepoStore"`
	Root   string `short:"r" long:"root" description:"the root of the store (repo clone dir for RepoStore, global path for MultiRepoStore, etc.)" default:".srclib-store"`
	Config string `long:"config" description:"(rarely used) JSON-encoded config for extra config, specific to each store type"`
}

var storeCmd StoreCmd

func (c *StoreCmd) Execute(args []string) error { return nil }

// store returns the store specified by StoreCmd's Type and Root
// options.
func (c *StoreCmd) store() (interface{}, error) {
	var fs rwvfs.FileSystem
	// Attempt to parse Root as a url, and fallback to creating an
	// OS file system if it isn't.
	if u, err := url.Parse(c.Root); err == nil && strings.HasSuffix(u.Host, "amazonaws.com") {
		fs = s3vfs.S3(u, nil)
	} else {
		fs = rwvfs.OS(c.Root)
	}
	switch c.Type {
	case "RepoStore":
		return store.NewFSRepoStore(fs), nil
	case "MultiRepoStore":
		var conf *store.FSMultiRepoStoreConf
		if c.Config != "" {
			// Only really allows configuring EvenlyDistributedRepoPaths right now.
			var conf2 struct {
				RepoPaths string
			}
			if err := json.Unmarshal([]byte(c.Config), &conf2); err != nil {
				return nil, fmt.Errorf("--config %q: %s", c.Config, err)
			}
			if conf2.RepoPaths == "EvenlyDistributedRepoPaths" {
				conf = &store.FSMultiRepoStoreConf{RepoPaths: &store.EvenlyDistributedRepoPaths{}}
			}
		}
		return store.NewFSMultiRepoStore(rwvfs.Walkable(fs), conf), nil
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

	RemoteBuildDataRepo string `long:"remote-build-data-repo" description:"the repo whose remote build data to import (defaults to '--repo' option value)"`
	RemoteBuildData     bool   `long:"remote-build-data" description:"import remote build data (not the local .srclib-cache build data)"`
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

	if c.RemoteBuildDataRepo == "" {
		c.RemoteBuildDataRepo = c.Repo
	}
	bdfs, label, err := getBuildDataFS(!c.RemoteBuildData, c.RemoteBuildDataRepo, c.CommitID)
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

func (c *StoreReposCmd) filters() []store.RepoFilter {
	var fs []store.RepoFilter
	if c.IDContains != "" {
		fs = append(fs, store.RepoFilterFunc(func(repo string) bool { return strings.Contains(repo, c.IDContains) }))
	}
	return fs
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

	repos, err := mrs.Repos(c.filters()...)
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

func (c *StoreVersionsCmd) filters() []store.VersionFilter {
	var fs []store.VersionFilter
	if c.Repo != "" {
		fs = append(fs, store.ByRepo(c.Repo))
	}
	if c.CommitIDPrefix != "" {
		fs = append(fs, store.VersionFilterFunc(func(version *store.Version) bool {
			return strings.HasPrefix(version.CommitID, c.CommitIDPrefix)
		}))
	}
	return fs
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

	versions, err := rs.Versions(c.filters()...)
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

func (c *StoreUnitsCmd) filters() []store.UnitFilter {
	var fs []store.UnitFilter
	if c.Type != "" && c.Name != "" {
		fs = append(fs, store.ByUnits(unit.ID2{Type: c.Type, Name: c.Name}))
	}
	if (c.Type != "" && c.Name == "") || (c.Type == "" && c.Name != "") {
		log.Fatal("must specify either both or neither of --type and --name (to filter by source unit)")
	}
	if c.CommitID != "" {
		fs = append(fs, store.ByCommitID(c.CommitID))
	}
	if c.Repo != "" {
		fs = append(fs, store.ByRepo(c.Repo))
	}
	if c.File != "" {
		fs = append(fs, store.ByFiles(path.Clean(c.File)))
	}
	return fs
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

	units, err := ts.Units(c.filters()...)
	if err != nil {
		return err
	}
	PrintJSON(units, "  ")
	return nil
}

type StoreDefsCmd struct {
	Repo           string `long:"repo"`
	Path           string `long:"path"`
	UnitType       string `long:"unit-type" `
	Unit           string `long:"unit"`
	File           string `long:"file"`
	FilePathPrefix string `long:"file-path-prefix"`
	CommitID       string `long:"commit"`

	NamePrefix string `long:"name-prefix"`

	Limit int `short:"n" long:"limit" description:"max results to return (0 for all)"`
}

func (c *StoreDefsCmd) filters() []store.DefFilter {
	var fs []store.DefFilter
	if c.UnitType != "" && c.Unit != "" {
		fs = append(fs, store.ByUnits(unit.ID2{Type: c.UnitType, Name: c.Unit}))
	}
	if (c.UnitType != "" && c.Unit == "") || (c.UnitType == "" && c.Unit != "") {
		log.Fatal("must specify either both or neither of --unit-type and --unit (to filter by source unit)")
	}
	if c.CommitID != "" {
		fs = append(fs, store.ByCommitID(c.CommitID))
	}
	if c.Repo != "" {
		fs = append(fs, store.ByRepo(c.Repo))
	}
	if c.Path != "" {
		fs = append(fs, store.ByDefPath(c.Path))
	}
	if c.File != "" {
		fs = append(fs, store.ByFiles(path.Clean(c.File)))
	}
	if c.FilePathPrefix != "" {
		fs = append(fs, store.ByFiles(path.Clean(c.FilePathPrefix)))
	}
	if c.NamePrefix != "" {
		fs = append(fs, store.DefFilterFunc(func(def *graph.Def) bool {
			return strings.HasPrefix(def.Name, c.NamePrefix)
		}))
	}
	if c.Limit != 0 {
		fs = append(fs, store.Limit(c.Limit))
	}
	return fs
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

	defs, err := us.Defs(c.filters()...)
	if err != nil {
		return err
	}
	PrintJSON(defs, "  ")
	return nil
}

type StoreRefsCmd struct {
	Repo     string `long:"repo"`
	UnitType string `long:"unit-type" `
	Unit     string `long:"unit"`
	File     string `long:"file"`
	CommitID string `long:"commit"`

	Start int `long:"start"`
	End   int `long:"end"`

	DefRepo     string `long:"def-repo"`
	DefUnitType string `long:"def-unit-type" `
	DefUnit     string `long:"def-unit"`
	DefPath     string `long:"def-path"`
}

func (c *StoreRefsCmd) filters() []store.RefFilter {
	var fs []store.RefFilter
	if c.UnitType != "" && c.Unit != "" {
		fs = append(fs, store.ByUnits(unit.ID2{Type: c.UnitType, Name: c.Unit}))
	}
	if (c.UnitType != "" && c.Unit == "") || (c.UnitType == "" && c.Unit != "") {
		log.Fatal("must specify either both or neither of --unit-type and --unit (to filter by source unit)")
	}
	if c.CommitID != "" {
		fs = append(fs, store.ByCommitID(c.CommitID))
	}
	if c.Repo != "" {
		fs = append(fs, store.ByRepo(c.Repo))
	}
	if c.File != "" {
		fs = append(fs, store.ByFiles(path.Clean(c.File)))
	}
	if c.Start != 0 {
		fs = append(fs, store.RefFilterFunc(func(ref *graph.Ref) bool {
			return ref.Start >= c.Start
		}))
	}
	if c.End != 0 {
		fs = append(fs, store.RefFilterFunc(func(ref *graph.Ref) bool {
			return ref.End <= c.End
		}))
	}
	if c.DefRepo != "" && c.DefUnitType != "" && c.DefUnit != "" && c.DefPath != "" {
		fs = append(fs, store.ByRefDef(graph.RefDefKey{
			DefRepo:     c.DefRepo,
			DefUnitType: c.DefUnitType,
			DefUnit:     c.DefUnit,
			DefPath:     c.DefPath,
		}))
	}
	if (c.DefRepo != "" || c.DefUnitType != "" || c.DefUnit != "" || c.DefPath != "") && (c.DefRepo == "" || c.DefUnitType == "" || c.DefUnit == "" || c.DefPath == "") {
		log.Fatal("must specify either all or neither of --def-repo, --def-unit-type, --def-unit, and --def-path (to filter by ref target def)")
	}
	return fs
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

	refs, err := us.Refs(c.filters()...)
	if err != nil {
		return err
	}
	PrintJSON(refs, "  ")
	return nil
}
