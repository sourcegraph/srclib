// +build off

package ruby

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/sourcegraph/srcscan"
	"sourcegraph.com/sourcegraph/dep"
	"github.com/sourcegraph/srclib/graph"
	"github.com/sourcegraph/srclib/repo"
	"sourcegraph.com/sourcegraph/util"
)

var skipGemPatterns = []string{
	"nokogiri", "pg", "rake", "mysql", "bcrypt-ruby", "debugger",
	"debugger-linecache", "debugger-ruby_core_source", "tzinfo",
}

const noBinaryDepsGemfileSuffix = ""

func (gg *gemGrapher) writeGemFileWithoutBinaryDeps() error {
	data, err := ioutil.ReadFile(gg.gemFilePath())
	if err != nil {
		return err
	}
	dataStr := string(data)
	lines := strings.Split(dataStr, "\n")

	var filteredLines []string
	for _, line := range lines {
		skip := false
		for _, pat := range skipGemPatterns {
			if strings.Contains(line, pat) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		filteredLines = append(filteredLines, line)
	}
	err = ioutil.WriteFile(gg.gemFilePath()+noBinaryDepsGemfileSuffix, []byte(strings.Join(filteredLines, "\n")), 0700)
	return err
}

func (g *rootGrapher) Dep(unit srcscan.Unit) (err error) {
	gg := g.gemGrapher(unit)

	err = g.ensureAnalyzeRubyCore()
	if err != nil {
		return
	}

	if gg.isRubyStdlib() {
		g.ctx.Log.Printf("Skipping dep phase for Ruby stdlib %s", g.ctx.Repo)
		return
	}

	if !gg.hasGemfile() {
		g.ctx.Log.Printf("Skipping dep phase because no Gemfile is found %s", g.ctx.Repo)
		return
	}

	g.ctx.Dep(&dep.Dep{ToRepoCloneURL: RubyMRICloneURL})

	if !gg.hasGemfile() {
		g.ctx.Log.Printf("No Gemfile found at %s; dep phase is complete", gg.gemFilePath())
		return
	}

	g.ctx.Log.Printf("Installing bundler dependencies")
	if gg.hasGemfile() {
		err = gg.writeGemFileWithoutBinaryDeps()
		if err != nil {
			return err
		}
	}
	cmd := rvmCommand("bundle", "install", "--gemfile=Gemfile"+noBinaryDepsGemfileSuffix, "--no-color")
	cmd.Dir = gg.absdir()
	cmd.Stderr, cmd.Stdout = g.ctx.Out, g.ctx.Out
	err = cmd.Run()
	if err != nil {
		g.ctx.Log.Printf("Warning: some gems failed to install (see log messages above): %s", err)
		err = nil
	}

	// Generate yard DB for deps.
	var gems []geminfo
	gems, err = gg.getBundleGems()
	if err != nil {
		return
	}

	for _, gem := range gems {
		if gem.name == "bundler" || gem.name == "nokogiri" || gem.name == "tzinfo" || gem.name == "rake" {
			// TODO: also skip the current source unit gem
			continue
		}
		var depCloneURL string
		depCloneURL, _, err = ResolveGem(gem.name)
		if err == nil {
			g.ctx.Dep(&dep.Dep{ToRepoCloneURL: depCloneURL})
		} else {
			g.ctx.Log.Printf("warn: Failed to resolve gem dependency %q to a clone URL: %s (continuing)", gem.name, err)
			err = nil
		}

		if !util.IsDir(gem.path) {
			return fmt.Errorf("Gem %q not found at path %q", gem.name, gem.path)
		}

		g.ctx.Log.Printf("Finished resolving gem dependency %q in %s", gem.name, gem.path)
	}

	return
}

func (g *rootGrapher) Build(unit srcscan.Unit) (err error) {
	return
}

func (g *rootGrapher) Analyze(unit srcscan.Unit) (err error) {
	gg := g.gemGrapher(unit)

	var basePath string
	if !gg.isRubyStdlib() {
		if gem, ok := unit.(*srcscan.RubyGem); ok {
			// this assumes that no 2 gems in the same repo have the same name
			if gem.Path() == "." {
				basePath = "gem"
			} else {
				basePath = "gems/" + gem.Name
			}
			gg.ctx.Symbol(&graph.Symbol{
				SymbolKey:    graph.SymbolKey{Path: graph.SymbolPath(basePath)},
				SpecificPath: gem.Name + " gem",
				Kind:         graph.Package,
				SpecificKind: "gem",
				Name:         gem.Name,
				File:         filepath.Join(gg.absdir(), gem.GemSpecFile),
				Exported:     true,
			})
			basePath += "/"
		}
	}

	var includePathList []string
	var gems []geminfo
	if gg.hasGemfile() {
		gems, err = gg.getBundleGems()
		if err != nil {
			return
		}

		for _, gem := range gems {
			// TODO(sqs): "lib" is not always the dir that the lib is under.
			includePathList = append(includePathList, filepath.Join(gem.path, "lib"))
		}
	}

	if !gg.isRubyStdlib() {
		includePathList = append(includePathList, RubyLibDir)
	}

	includePaths := strings.Join(includePathList, ":")

	var allRubyFiles []string
	outputPrefix := filepath.Join(gg.ctx.WorkDir, strings.TrimPrefix(fmt.Sprintf("%T", unit), "*"), unit.Path(), "out")
	err = os.MkdirAll(filepath.Dir(outputPrefix), 0700)
	if err != nil {
		return err
	}

	sourcePath := gg.absdir()
	args := []string{"-Xmx4G", "-cp", RubySonarPath, "org.yinwang.rubysonar.JSONDump", sourcePath, outputPrefix, includePaths}

	for _, f := range gg.unitSrcFiles() {
		abspath := filepath.Join(gg.absdir(), f)
		args = append(args, abspath)
		allRubyFiles = append(allRubyFiles, abspath)
	}
	for _, f := range gg.unitTestFiles() {
		abspath := filepath.Join(gg.absdir(), f)
		args = append(args, abspath)
		allRubyFiles = append(allRubyFiles, abspath)
	}

	cmd := rvmCommand("java", args...)
	cmd.Dir = g.ctx.Dir
	//	g.ctx.Log.Printf("Running: %v", cmd.Args)
	cmd.Stderr = g.ctx.Out
	cmd.Stdout = g.ctx.Out

	err = cmd.Run()
	if err != nil {
		return
	}

	var readJSONFile = func(file string, v interface{}) error {
		f, err := os.Open(file)
		if err != nil {
			return err
		}
		defer f.Close()
		return json.NewDecoder(f).Decode(v)
	}

	var rubysonarData struct {
		Objects    []*rubyObject
		References []*rubyRef
	}
	err = readJSONFile(outputPrefix+"-sym", &rubysonarData.Objects)
	if err != nil {
		return
	}
	err = readJSONFile(outputPrefix+"-ref", &rubysonarData.References)
	if err != nil {
		return
	}

	// Eliminate docstrings from rubysonar objects so that we use the YARD
	// docstring instead.
	for _, o := range rubysonarData.Objects {
		o.Docstring = ""
	}

	seensym := make(map[graph.SymbolKey]graph.Symbol)
	seendoc := make(map[graph.SymbolKey]struct{})
	for _, rubyObj := range rubysonarData.Objects {
		sym := rubyObj.toSymbol()
		sym.Path = graph.SymbolPath(basePath + string(sym.Path))

		if _, seen := seensym[sym.SymbolKey]; !seen {
			seensym[sym.SymbolKey] = *sym

			if !gg.isRubyStdlib() {
				// Only emit symbols that were defined first in one of the files we're
				// analyzing. Otherwise, we emit duplicate symbols when a class or
				// module is reopened. TODO(sqs): might not be necessary if we suppress
				// these at the ruby level.
				found := false
				for _, f := range allRubyFiles {
					if sym.File == f {
						found = true
						break
					}
				}
				if !found {
					if !strings.Contains(os.Getenv("SG_RUBY_LOG"), "-reopen") {
						//					g.ctx.Log.Printf("Skipping symbol at path %s whose first definition was in a different source unit at %s (reopened class or module?)", sym.Path, sym.File)
					}
					continue
				}
			}

			gg.ctx.Symbol(sym)
		}
	}

	if !util.ParseBool(os.Getenv("SG_SKIP_YARDOC")) {
		yardRubyObjs, err := g.yardObjects(allRubyFiles)
		if err != nil {
			return fmt.Errorf("error running YARD: %s", err)
		}

		for _, rubyObj := range yardRubyObjs {
			sym := rubyObj.toSymbol()
			sym.Path = graph.SymbolPath(basePath + string(sym.Path))
			if len(sym.Path) > 1000 {
				continue
			}

			if _, seen := seendoc[sym.SymbolKey]; !seen {
				if rubyObj.Docstring != "" {
					seendoc[sym.SymbolKey] = struct{}{}
					gg.ctx.Doc(&graph.Doc{
						SymbolKey: sym.SymbolKey,
						Body:      rubyObj.Docstring,
					})
				}
			}
		}
	}

	printedGemResolutionErr := make(map[string]struct{})
	seenref := make(map[graph.Ref]struct{})

	for _, rubyRef := range rubysonarData.References {
		ref, targetOriginFile := rubyRef.toRef()
		if _, seen := seenref[*ref]; seen {
			continue
		}
		seenref[*ref] = struct{}{}

		// Resolve targetOriginFile to either 1) a gem, 2) the Ruby core/stdlib,
		// or 3) this Ruby source unit.
		if strings.HasPrefix(targetOriginFile, gg.absdir()) {
			// internal ref to this source unit.
			ref.SymbolPath = graph.SymbolPath(basePath + string(ref.SymbolPath))
		} else {
			isRefToGem := false
			for _, gem := range gems {
				if !strings.HasPrefix(targetOriginFile, gem.path) {
					continue
				}

				// external reference to a gem
				isRefToGem = true

				var cloneURL, gemPath string
				cloneURL, gemPath, err = ResolveGem(gem.name)
				if err == nil {
					ref.SymbolRepo = repo.MakeURI(cloneURL)
					var pathPrefix string
					if gemPath == "" {
						pathPrefix = "gem"
					} else {
						pathPrefix = "gems/" + gemPath
					}
					ref.SymbolPath = graph.SymbolPath(pathPrefix + "/" + string(ref.SymbolPath))
				} else {
					if _, alreadyPrinted := printedGemResolutionErr[gem.name]; !alreadyPrinted {
						g.ctx.Log.Printf("want: Failed to resolve gem dependency %q to clone URL: %s (continuing, not emitting reference, and suppressing future identical log messages)", gem.name, err)
						printedGemResolutionErr[gem.name] = struct{}{}
					}
					err = nil
					continue
				}
			}

			if !isRefToGem {
				// ref to Ruby stdlib or core
				ref.SymbolRepo = repo.MakeURI(RubyMRICloneURL)
			}
		}

		if len(ref.SymbolPath) > 1000 {
			continue
		}

		gg.ctx.Ref(ref)
	}

	return
}

type rubyObject struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	Module     string `json:"module"`
	Type       string `json:"kind"`
	File       string `json:"file"`
	IdentStart int    `json:"identStart"`
	IdentEnd   int    `json:"identEnd"`
	DefStart   int    `json:"defStart"`
	DefEnd     int    `json:"defEnd"`
	Docstring  string `json:"docstring"`
	TypeExpr   string `json:"signature"`
	Exported   bool   `json:"exported"`
}

func (s *rubyObject) toSymbol() *graph.Symbol {
	return &graph.Symbol{
		SymbolKey:    graph.SymbolKey{Path: rubyPathToSymbolPath(s.Path)},
		SpecificPath: s.Path,
		Kind:         rubyObjectTypeMap[strings.ToLower(s.Type)],
		SpecificKind: s.Type,
		Name:         s.Name,
		Exported:     s.Exported,
		Callable:     s.Type == "method",
		File:         s.File,
		IdentStart:   s.IdentStart,
		IdentEnd:     s.IdentEnd,
		DefStart:     s.DefStart,
		DefEnd:       s.DefEnd,
		TypeExpr:     s.TypeExpr,
	}
}

var rubyObjectTypeMap = map[string]graph.SymbolKind{
	"module":    graph.Module,
	"class":     graph.Type,
	"method":    graph.Func,
	"variable":  graph.Var,
	"attribute": graph.Var,
	"constant":  graph.Const,
}

type rubyRef struct {
	Target           string `json:"sym"`
	TargetOriginFile string `json:"symFile"`
	Kind             string `json:"kind"`
	File             string `json:"file"`
	Start            int    `json:"start"`
	End              int    `json:"end"`
	Builtin          bool   `json:"builtin"`
}

func (r *rubyRef) toRef() (ref *graph.Ref, targetOrigin string) {
	if r.Kind == "" {
		r.Kind = "ident"
	}
	return &graph.Ref{
		SymbolPath: rubyPathToSymbolPath(r.Target),
		Kind:       graph.RefKind(r.Kind),
		Location: graph.Location{
			File:  r.File,
			Start: r.Start,
			End:   r.End,
		},
	}, r.TargetOriginFile
}

func rubyPathToSymbolPath(path string) graph.SymbolPath {
	p := strings.Replace(strings.Replace(strings.Replace(strings.Replace(path, ".rb", "_rb", -1), "::", "/", -1), "#", "/$methods/", -1), ".", "/$classmethods/", -1)
	return graph.SymbolPath(strings.TrimPrefix(p, "/"))
}
