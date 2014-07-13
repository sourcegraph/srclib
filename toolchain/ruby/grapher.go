package ruby

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"

	"strings"

	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/container"
	"sourcegraph.com/sourcegraph/srcgraph/graph"
	"sourcegraph.com/sourcegraph/srcgraph/grapher2"
	"sourcegraph.com/sourcegraph/srcgraph/repo"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
)

func init() {
	grapher2.Register(&RubyGem{}, grapher2.DockerGrapher{DefaultRubyVersion})
	grapher2.Register(&RubyLib{}, grapher2.DockerGrapher{DefaultRubyVersion})
}

const (
	RubyStdlibYARDocDir = "/tmp/ruby-stdlib-yardoc"
)

func (v *Ruby) BuildGrapher(dir string, unit unit.SourceUnit, c *config.Repository) (*container.Command, error) {
	rubyConfig := v.rubyConfig(c)

	const (
		containerDir = "/tmp/rubygem"
	)
	rubySrcDir := fmt.Sprintf("/usr/local/rvm/src/ruby-%s", v.Version)

	gemDir := filepath.Join(containerDir, unit.RootDir())

	dockerfile_, err := v.baseDockerfile()
	if err != nil {
		return nil, err
	}

	dockerfile := bytes.NewBuffer(dockerfile_)

	// Set up YARD
	fmt.Fprintln(dockerfile, "\n# Set up YARD")
	fmt.Fprintln(dockerfile, "RUN apt-get install -qy git")
	fmt.Fprintln(dockerfile, "RUN git clone git://github.com/sourcegraph/yard.git /yard && cd /yard && git checkout cf7d77784dfddd11a1a76aea705271178e1d369e")
	fmt.Fprintln(dockerfile, "RUN cd /yard && rvm all do bundle && rvm all do gem install asciidoctor rdoc --no-rdoc --no-ri")

	if !rubyConfig.OmitStdlib {
		// Process the Ruby stdlib.
		fmt.Fprintf(dockerfile, "\n# Process the Ruby stdlib (version %s)\n", v.Version)
		fmt.Fprintf(dockerfile, "RUN rvm fetch %s\n", v.Version)
		fmt.Fprintf(dockerfile, "RUN rvm all do /yard/bin/yard doc -c %s -n %s/*.c '%s/lib/**/*.rb'\n", RubyStdlibYARDocDir, rubySrcDir, rubySrcDir)
	}

	cont := container.Container{
		Dockerfile: dockerfile.Bytes(),
		AddDirs:    [][2]string{{dir, containerDir}},
		Dir:        gemDir,
		PreCmdDockerfile: []byte(`
WORKDIR ` + gemDir + `
# Remove common binary deps from Gemfile (hacky)
RUN if [ -e Gemfile ]; then sed -i '/\(pg\|nokigiri\|rake\|mysql\|bcrypt-ruby\|debugger\|debugger-linecache\|debugger-ruby_core_source\|tzinfo\)/d' Gemfile; fi
RUN if [ -e Gemfile ]; then rvm all do bundle install --no-color; fi
RUN if [ -e Gemfile ]; then rvm all do /yard/bin/yard bundle --debug; fi
`),
		Cmd: []string{"bash", "-c", "rvm all do /yard/bin/yard condense -c " + RubyStdlibYARDocDir + " --load-yardoc-files `test -e Gemfile && rvm all do /yard/bin/yard bundle --list | cut -f 2 | paste -sd ,`,/dev/null " + strings.Join(unit.Paths(), " ")},
	}

	cmd := container.Command{
		Container: cont,
		Transform: func(orig []byte) ([]byte, error) {
			var data *yardocCondenseOutput
			err := json.Unmarshal(orig, &data)
			if err != nil {
				return nil, err
			}

			// Convert data to srcgraph format.
			o2, err := v.convertGraphData(data, c)
			if err != nil {
				return nil, err
			}

			return json.Marshal(o2)
		},
	}

	return &cmd, nil
}

type yardocCondenseOutput struct {
	Objects    []*rubyObject
	References []*rubyRef
}

// convertGraphData converts graph data from `yard condense` output format to srcgraph
// format.
func (v *Ruby) convertGraphData(ydoc *yardocCondenseOutput, c *config.Repository) (*grapher2.Output, error) {
	o := grapher2.Output{
		Symbols: make([]*graph.Symbol, 0, len(ydoc.Objects)),
		Refs:    make([]*graph.Ref, 0, len(ydoc.References)),
	}

	seensym := make(map[graph.SymbolKey]graph.Symbol)

	type seenRefKey struct {
		graph.RefSymbolKey
		File       string
		Start, End int
	}
	seenref := make(map[seenRefKey]struct{})

	for _, rubyObj := range ydoc.Objects {
		sym, err := rubyObj.toSymbol()
		if err != nil {
			return nil, err
		}

		if prevSym, seen := seensym[sym.SymbolKey]; seen {
			log.Printf("Skipping already seen symbol %+v -- other def is %+v", prevSym, sym)
			continue
		}
		seensym[sym.SymbolKey] = *sym

		// TODO(sqs) TODO(ruby): implement this
		// if !gg.isRubyStdlib() {
		// 	// Only emit symbols that were defined first in one of the files we're
		// 	// analyzing. Otherwise, we emit duplicate symbols when a class or
		// 	// module is reopened. TODO(sqs): might not be necessary if we suppress
		// 	// these at the ruby level.
		// 	found := false
		// 	for _, f := range allRubyFiles {
		// 		if sym.File == f {
		// 			found = true
		// 			break
		// 		}
		// 	}
		// 	if !found {
		// 		log.Printf("Skipping symbol at path %s whose first definition was in a different source unit at %s (reopened class or module?)", sym.Path, sym.File)
		// 		continue
		// 	}
		// }

		o.Symbols = append(o.Symbols, sym)

		if rubyObj.Docstring != "" {
			o.Docs = append(o.Docs, &graph.Doc{
				SymbolKey: sym.SymbolKey,
				Format:    "text/html",
				Data:      rubyObj.Docstring,
				File:      rubyObj.File,
			})
		}

		// Defs parsed from C code have a name_range (instead of a ref with
		// decl_ident). Emit those as refs here.
		if rubyObj.NameStart != 0 || rubyObj.NameEnd != 0 {
			nameRef := &graph.Ref{
				SymbolPath: sym.Path,
				Def:        true,
				File:       sym.File,
				Start:      rubyObj.NameStart,
				End:        rubyObj.NameEnd,
			}
			seenref[seenRefKey{nameRef.RefSymbolKey(), nameRef.File, nameRef.Start, nameRef.End}] = struct{}{}
			o.Refs = append(o.Refs, nameRef)
		}
	}

	printedGemResolutionErr := make(map[string]struct{})

	for _, rubyRef := range ydoc.References {
		ref, depGemName := rubyRef.toRef()

		// Determine the referenced symbol's repo.
		if depGemName == StdlibGemNameSentinel {
			// Ref to stdlib.
			ref.SymbolRepo = repo.MakeURI(v.StdlibCloneURL)
			ref.SymbolUnit = "."
			ref.SymbolUnitType = unit.Type(&RubyLib{})
		} else if depGemName != "" {
			// Ref to another gem.
			cloneURL, err := ResolveGem(depGemName)
			if err != nil {
				if _, alreadyPrinted := printedGemResolutionErr[depGemName]; !alreadyPrinted {
					log.Printf("Warning: Failed to resolve gem dependency %q to clone URL: %s (continuing, not emitting reference, and suppressing future identical log messages)", depGemName, err)
					printedGemResolutionErr[depGemName] = struct{}{}
				}
				continue
			}
			ref.SymbolRepo = repo.MakeURI(cloneURL)
			ref.SymbolUnit = depGemName
		} else if depGemName == "" {
			// Internal ref to this gem.
		}

		seenKey := seenRefKey{ref.RefSymbolKey(), ref.File, ref.Start, ref.End}
		if _, seen := seenref[seenKey]; seen {
			log.Printf("Already saw ref key %v; skipping.", seenKey)
			continue
		}
		seenref[seenKey] = struct{}{}

		o.Refs = append(o.Refs, ref)
	}

	return &o, nil
}

type rubyObject struct {
	Name       string
	Path       string
	Module     string
	Type       string
	File       string
	Exported   bool
	DefStart   int `json:"def_start"`
	DefEnd     int `json:"def_end"`
	NameStart  int `json:"name_start"`
	NameEnd    int `json:"name_end"`
	Docstring  string
	Signature  string `json:"signature"`
	TypeString string `json:"type_string"`
	ReturnType string `json:"return_type"`
}

type SymbolData struct {
	RubyKind   string
	TypeString string
	Module     string
	RubyPath   string
	Signature  string
	ReturnType string
}

func (s *SymbolData) isLocalVar() bool {
	return strings.Contains(s.RubyPath, ">_local_")
}

func (s *rubyObject) toSymbol() (*graph.Symbol, error) {
	sym := &graph.Symbol{
		SymbolKey: graph.SymbolKey{Path: rubyPathToSymbolPath(s.Path)},
		TreePath:  rubyPathToTreePath(s.Path),
		Kind:      rubyObjectTypeMap[s.Type],
		Name:      s.Name,
		Exported:  s.Exported,
		File:      s.File,
		DefStart:  s.DefStart,
		DefEnd:    s.DefEnd,
		Test:      strings.Contains(s.File, "_test.rb") || strings.Contains(s.File, "_spec.rb") || strings.Contains(s.File, "test/") || strings.Contains(s.File, "spec/"),
	}

	d := SymbolData{
		RubyKind:   s.Type,
		TypeString: s.TypeString,
		Signature:  s.Signature,
		Module:     s.Module,
		RubyPath:   s.Path,
		ReturnType: s.ReturnType,
	}
	var err error
	sym.Data, err = json.Marshal(d)
	if err != nil {
		return nil, err
	}

	return sym, nil
}

var rubyObjectTypeMap = map[string]graph.SymbolKind{
	"method":           graph.Func,
	"constant":         graph.Const,
	"class":            graph.Type,
	"module":           graph.Module,
	"localvariable":    graph.Var,
	"instancevariable": graph.Var,
	"classvariable":    graph.Var,
}

type rubyRef struct {
	Target                 string
	TargetOriginYardocFile string `json:"target_origin_yardoc_file"`
	Kind                   string
	File                   string
	Start                  int
	End                    int
}

func (r *rubyRef) toRef() (ref *graph.Ref, targetOrigin string) {
	return &graph.Ref{
		SymbolPath: rubyPathToSymbolPath(r.Target),
		Def:        r.Kind == "decl_ident",
		File:       r.File,
		Start:      r.Start,
		End:        r.End,
	}, getGemNameFromGemYardocFile(r.TargetOriginYardocFile)
}

func rubyPathToSymbolPath(path string) graph.SymbolPath {
	p := strings.Replace(strings.Replace(strings.Replace(strings.Replace(strings.Replace(path, ".rb", "_rb", -1), "::", "/", -1), "#", "/$methods/", -1), ".", "/$classmethods/", -1), ">", "@", -1)
	return graph.SymbolPath(strings.TrimPrefix(p, "/"))
}

func rubyPathToTreePath(path string) graph.TreePath {
	path = strings.Replace(strings.Replace(strings.Replace(strings.Replace(strings.Replace(path, ".rb", "_rb", -1), "::", "/", -1), "#", "/", -1), ".", "/", -1), ">", "/", -1)
	parts := strings.Split(path, "/")
	var meaningfulParts []string
	for _, p := range parts {
		if strings.HasPrefix(p, "_local_") || p == "" {
			// Strip out path components that exist solely to make this path
			// unique and are not semantically meaningful.
			meaningfulParts = append(meaningfulParts, p)
		}
	}
	return graph.TreePath(strings.Join(meaningfulParts, "/"))
}
