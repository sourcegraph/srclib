package src

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/kr/fs"
	"sourcegraph.com/sourcegraph/go-sourcegraph/sourcegraph"
	"sourcegraph.com/sourcegraph/srclib/buildstore"
	"sourcegraph.com/sourcegraph/srclib/config"
	"sourcegraph.com/sourcegraph/srclib/dep"
	"sourcegraph.com/sourcegraph/srclib/graph"
	"sourcegraph.com/sourcegraph/srclib/grapher"
	"sourcegraph.com/sourcegraph/srclib/plan"
	"sourcegraph.com/sourcegraph/srclib/unit"
)

func init() {
	c, err := CLI.AddCommand("api",
		"API",
		"",
		&apiCmd,
	)
	if err != nil {
		log.Fatal(err)
	}

	_, err = c.AddCommand("describe",
		"display documentation for the def under the cursor",
		"Returns information about the definition referred to by the cursor's current position in a file.",
		&apiDescribeCmd,
	)
	if err != nil {
		log.Fatal(err)
	}

	_, err = c.AddCommand("list",
		"list all refs in a given file",
		"Return a list of all references that are in the current file.",
		&apiListCmd,
	)
	if err != nil {
		log.Fatal(err)
	}

	_, err = c.AddCommand("deps",
		"list all resolved and unresolved dependencies",
		"Return a list of all resolved and unresolved dependencies that are in the current repository.",
		&apiDepsCmd,
	)
	if err != nil {
		log.Fatal(err)
	}

	_, err = c.AddCommand("units",
		"list all source unit information",
		"Return a list of all source units that are in the current repository.",
		&apiUnitsCmd,
	)
	if err != nil {
		log.Fatal(err)
	}
}

type APICmd struct{}

var apiCmd APICmd

func (c *APICmd) Execute(args []string) error { return nil }

type APIDescribeCmd struct {
	File      string `long:"file" required:"yes" value-name:"FILE"`
	StartByte int    `long:"start-byte" required:"yes" value-name:"BYTE"`

	NoExamples bool `long:"no-examples" describe:"don't show examples from Sourcegraph.com"`
}

type APIListCmd struct {
	File string `long:"file" required:"yes" value-name:"FILE"`
}

type APIDepsCmd struct {
	Args struct {
		Dir Directory `name:"DIR" default:"." description:"root directory of target project"`
	} `positional-args:"yes"`
}

type APIUnitsCmd struct {
	Args struct {
		Dir Directory `name:"DIR" default:"." description:"root directory of target project"`
	} `positional-args:"yes"`
}

var apiDescribeCmd APIDescribeCmd
var apiListCmd APIListCmd
var apiDepsCmd APIDepsCmd
var apiUnitsCmd APIUnitsCmd

// Invokes the build process on the given repository
func ensureBuild(buildStore buildstore.RepoBuildStore, repo *Repo) error {
	configOpt := config.Options{
		Repo:   repo.URI(),
		Subdir: ".",
	}
	toolchainExecOpt := ToolchainExecOpt{ExeMethods: "program"}

	// Config repository if not yet built.
	exists, err := buildstore.BuildDataExistsForCommit(buildStore, repo.CommitID)
	if err != nil {
		return err
	}
	if !exists {
		configCmd := &ConfigCmd{
			Options:          configOpt,
			ToolchainExecOpt: toolchainExecOpt,
			w:                os.Stderr,
		}
		if err := configCmd.Execute(nil); err != nil {
			return err
		}
	}

	// Always re-make.
	//
	// TODO(sqs): optimize this
	makeCmd := &MakeCmd{
		Options:          configOpt,
		ToolchainExecOpt: toolchainExecOpt,
	}
	if err := makeCmd.Execute(nil); err != nil {
		return err
	}

	return nil
}

// Get a list of all source units that contain the given file
func getSourceUnitsWithFile(buildStore buildstore.RepoBuildStore, repo *Repo, filename string) ([]*unit.SourceUnit, error) {
	filename = filepath.Clean(filename)

	// TODO(sqs): This whole lookup is totally inefficient. The storage format
	// is not optimized for lookups.

	// Find all source unit definition files.
	var unitFiles []string
	unitSuffix := buildstore.DataTypeSuffix(unit.SourceUnit{})
	commitFS := buildStore.Commit(repo.CommitID)
	w := fs.WalkFS(".", commitFS)
	for w.Step() {
		if strings.HasSuffix(w.Path(), unitSuffix) {
			unitFiles = append(unitFiles, w.Path())
		}
	}

	// Find which source units the file belongs to.
	var units []*unit.SourceUnit
	for _, unitFile := range unitFiles {
		var u *unit.SourceUnit
		f, err := commitFS.Open(unitFile)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		if err := json.NewDecoder(f).Decode(&u); err != nil {
			return nil, fmt.Errorf("%s: %s", unitFile, err)
		}
		for _, f2 := range u.Files {
			if filepath.Clean(f2) == filename {
				units = append(units, u)
				break
			}
		}
	}

	return units, nil
}

func (c *APIListCmd) Execute(args []string) error {
	var err error
	c.File, err = filepath.Abs(c.File)
	if err != nil {
		return err
	}

	repo, err := OpenRepo(filepath.Dir(c.File))
	if err != nil {
		return err
	}

	c.File, err = filepath.Rel(repo.RootDir, c.File)
	if err != nil {
		return err
	}

	if err := os.Chdir(repo.RootDir); err != nil {
		return err
	}

	buildStore, err := buildstore.LocalRepo(repo.RootDir)
	if err != nil {
		return err
	}
	commitFS := buildStore.Commit(repo.CommitID)

	if err := ensureBuild(buildStore, repo); err != nil {
		if err := buildstore.RemoveAllDataForCommit(buildStore, repo.CommitID); err != nil {
			log.Println(err)
		}
		return err
	}

	units, err := getSourceUnitsWithFile(buildStore, repo, c.File)
	if err != nil {
		return err
	}

	if GlobalOpt.Verbose {
		if len(units) > 0 {
			ids := make([]string, len(units))
			for i, u := range units {
				ids[i] = string(u.ID())
			}
			log.Printf("File %s is in %d source units %v.", c.File, len(units), ids)
		} else {
			log.Printf("File %s is not in any source units.", c.File)
		}
	}

	// Find the ref(s) at the character position.
	var refs []*graph.Ref
	for _, u := range units {
		var g grapher.Output
		graphFile := plan.SourceUnitDataFilename("graph", u)
		f, err := commitFS.Open(graphFile)
		if err != nil {
			return err
		}
		defer f.Close()
		if err := json.NewDecoder(f).Decode(&g); err != nil {
			return fmt.Errorf("%s: %s", graphFile, err)
		}
		for _, ref := range g.Refs {
			if c.File == ref.File {
				refs = append(refs, ref)
			}
		}
	}

	if err := json.NewEncoder(os.Stdout).Encode(refs); err != nil {
		return err
	}
	return nil
}

func (c *APIDescribeCmd) Execute(args []string) error {
	var err error
	c.File, err = filepath.Abs(c.File)
	if err != nil {
		return err
	}

	repo, err := OpenRepo(filepath.Dir(c.File))
	if err != nil {
		return err
	}

	c.File, err = filepath.Rel(repo.RootDir, c.File)
	if err != nil {
		return err
	}

	if err := os.Chdir(repo.RootDir); err != nil {
		return err
	}

	buildStore, err := buildstore.LocalRepo(repo.RootDir)
	if err != nil {
		return err
	}
	commitFS := buildStore.Commit(repo.CommitID)

	if err := ensureBuild(buildStore, repo); err != nil {
		if err := buildstore.RemoveAllDataForCommit(buildStore, repo.CommitID); err != nil {
			log.Println(err)
		}
		return err
	}

	units, err := getSourceUnitsWithFile(buildStore, repo, c.File)
	if err != nil {
		return err
	}

	if GlobalOpt.Verbose {
		if len(units) > 0 {
			ids := make([]string, len(units))
			for i, u := range units {
				ids[i] = string(u.ID())
			}
			log.Printf("Position %s:%d is in %d source units %v.", c.File, c.StartByte, len(units), ids)
		} else {
			log.Printf("Position %s:%d is not in any source units.", c.File, c.StartByte)
		}
	}

	// Find the ref(s) at the character position.
	var ref *graph.Ref
	var nearbyRefs []*graph.Ref // Find nearby refs to help with debugging.
OuterLoop:
	for _, u := range units {
		var g grapher.Output
		graphFile := plan.SourceUnitDataFilename("graph", u)
		f, err := commitFS.Open(graphFile)
		if err != nil {
			return err
		}
		defer f.Close()
		if err := json.NewDecoder(f).Decode(&g); err != nil {
			return fmt.Errorf("%s: %s", graphFile, err)
		}
		for _, ref2 := range g.Refs {
			if c.File == ref2.File {
				if c.StartByte >= ref2.Start && c.StartByte <= ref2.End {
					ref = ref2
					if ref.DefUnit == "" {
						ref.DefUnit = u.Name
					}
					if ref.DefUnitType == "" {
						ref.DefUnitType = u.Type
					}
					break OuterLoop
				} else if GlobalOpt.Verbose && abs(ref2.Start-c.StartByte) < 25 {
					nearbyRefs = append(nearbyRefs, ref2)
				}
			}
		}
	}

	if ref == nil {
		if GlobalOpt.Verbose {
			log.Printf("No ref found at %s:%d.", c.File, c.StartByte)

			if len(nearbyRefs) > 0 {
				log.Printf("However, nearby refs were found in the same file:")
				for _, nref := range nearbyRefs {
					log.Printf("Ref at bytes %d-%d to %v", nref.Start, nref.End, nref.DefKey())
				}
			}

			f, err := os.Open(c.File)
			if err == nil {
				defer f.Close()
				b, err := ioutil.ReadAll(f)
				if err != nil {
					log.Fatalf("Error reading source file: %s.", err)
				}
				start := c.StartByte
				if start < 0 || start > len(b)-1 {
					log.Fatalf("Start byte %d is out of file bounds.", c.StartByte)
				}
				end := c.StartByte + 50
				if end > len(b)-1 {
					end = len(b) - 1
				}
				log.Printf("Surrounding source is:\n\n%s", b[start:end])
			} else {
				log.Printf("Error opening source file to show surrounding source: %s.", err)
			}
		}
		fmt.Println(`{}`)
		return nil
	}

	if ref.DefRepo == "" {
		ref.DefRepo = repo.URI()
	}

	var resp struct {
		Def      *sourcegraph.Def
		Examples []*sourcegraph.Example
	}

	// Now find the def for this ref.
	defInCurrentRepo := ref.DefRepo == repo.URI()
	if defInCurrentRepo {
		// Def is in the current repo.
		var g grapher.Output
		graphFile := plan.SourceUnitDataFilename("graph", &unit.SourceUnit{Name: ref.DefUnit, Type: ref.DefUnitType})
		f, err := commitFS.Open(graphFile)
		if err != nil {
			return err
		}
		defer f.Close()
		if err := json.NewDecoder(f).Decode(&g); err != nil {
			return fmt.Errorf("%s: %s", graphFile, err)
		}
		for _, def2 := range g.Defs {
			if def2.Path == ref.DefPath {
				resp.Def = &sourcegraph.Def{Def: *def2}
				break
			}
		}
		if resp.Def != nil {
			for _, doc := range g.Docs {
				if doc.Path == ref.DefPath {
					resp.Def.DocHTML = doc.Data
				}
			}

			// If Def is in the current Repo, transform that path to be an absolute path
			resp.Def.File = filepath.Join(repo.RootDir, resp.Def.File)
		}
		if resp.Def == nil && GlobalOpt.Verbose {
			log.Printf("No definition found with path %q in unit %q type %q.", ref.DefPath, ref.DefUnit, ref.DefUnitType)
		}
	}

	spec := sourcegraph.DefSpec{
		Repo:     string(ref.DefRepo),
		UnitType: ref.DefUnitType,
		Unit:     ref.DefUnit,
		Path:     string(ref.DefPath),
	}

	apiclient := NewAPIClientWithAuthIfPresent()

	var wg sync.WaitGroup

	if resp.Def == nil {
		// Def is not in the current repo. Try looking it up using the
		// Sourcegraph API.
		wg.Add(1)
		go func() {
			defer wg.Done()
			var err error
			resp.Def, _, err = apiclient.Defs.Get(spec, &sourcegraph.DefGetOptions{Doc: true})
			if err != nil && GlobalOpt.Verbose {
				log.Printf("Couldn't fetch definition %v: %s.", spec, err)
			}
		}()
	}

	if !c.NoExamples {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var err error
			resp.Examples, _, err = apiclient.Defs.ListExamples(spec, &sourcegraph.DefListExamplesOptions{
				Formatted:   true,
				ListOptions: sourcegraph.ListOptions{PerPage: 4},
			})
			if err != nil && GlobalOpt.Verbose {
				log.Printf("Couldn't fetch examples for %v: %s.", spec, err)
			}
		}()
	}

	wg.Wait()

	if err := json.NewEncoder(os.Stdout).Encode(resp); err != nil {
		return err
	}
	return nil
}

func abs(n int) int {
	if n < 0 {
		return -1 * n
	}
	return n
}

func (c *APIDepsCmd) Execute(args []string) error {
	var err error

	repo, err := OpenRepo(filepath.Dir(string(c.Args.Dir)))
	if err != nil {
		return err
	}

	if err := os.Chdir(repo.RootDir); err != nil {
		return err
	}

	buildStore, err := buildstore.LocalRepo(repo.RootDir)
	if err != nil {
		return err
	}
	commitFS := buildStore.Commit(repo.CommitID)

	exists, err := buildstore.BuildDataExistsForCommit(buildStore, repo.CommitID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("No build data found. Try running `src config` first.")
	}

	var depSlice []*dep.Resolution
	// TODO: Make DataTypeSuffix work with type of depSlice
	depSuffix := buildstore.DataTypeSuffix([]*dep.ResolvedDep{})
	depCache := make(map[string]struct{})
	foundDepresolve := false
	w := fs.WalkFS(".", commitFS)
	for w.Step() {
		depfile := w.Path()
		if strings.HasSuffix(depfile, depSuffix) {
			foundDepresolve = true
			var deps []*dep.Resolution
			f, err := commitFS.Open(depfile)
			if err != nil {
				return err
			}
			defer f.Close()
			if err := json.NewDecoder(f).Decode(&deps); err != nil {
				return fmt.Errorf("%s: %s", depfile, err)
			}
			for _, d := range deps {
				key := d.KeyId()
				if _, ok := depCache[key]; !ok {
					depCache[key] = struct{}{}
					depSlice = append(depSlice, d)
				}
			}
		}
	}

	if foundDepresolve == false {
		return errors.New("No dependency information found. Try running `src config` first.")
	}

	return json.NewEncoder(os.Stdout).Encode(depSlice)
}

func (c *APIUnitsCmd) Execute(args []string) error {
	var err error

	repo, err := OpenRepo(filepath.Dir(string(c.Args.Dir)))
	if err != nil {
		return err
	}

	if err := os.Chdir(repo.RootDir); err != nil {
		return err
	}

	buildStore, err := buildstore.LocalRepo(repo.RootDir)
	if err != nil {
		return err
	}
	commitFS := buildStore.Commit(repo.CommitID)

	exists, err := buildstore.BuildDataExistsForCommit(buildStore, repo.CommitID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("No build data found. Try running `src config` first.")
	}

	var unitSlice []unit.SourceUnit
	unitSuffix := buildstore.DataTypeSuffix(unit.SourceUnit{})
	foundUnit := false
	w := fs.WalkFS(".", commitFS)
	for w.Step() {
		unitFile := w.Path()
		if strings.HasSuffix(unitFile, unitSuffix) {
			var unit unit.SourceUnit
			foundUnit = true
			f, err := commitFS.Open(unitFile)
			if err != nil {
				return err
			}
			defer f.Close()
			if err := json.NewDecoder(f).Decode(&unit); err != nil {
				return fmt.Errorf("%s: %s", unitFile, err)
			}
			unitSlice = append(unitSlice, unit)
		}
	}

	if foundUnit == false {
		return errors.New("No source units found. Try running `src config` first.")
	}

	return json.NewEncoder(os.Stdout).Encode(unitSlice)
}
