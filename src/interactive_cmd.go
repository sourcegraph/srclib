package src

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strings"

	"sourcegraph.com/sourcegraph/srclib/buildstore"
	"sourcegraph.com/sourcegraph/srclib/graph"

	"github.com/nemith/goline"
	"github.com/peterh/liner"
)

func init() {
	interactiveGroup, err := CLI.AddCommand("interactive",
		"interactive REPL for build data",
		"The interactive (i) command is a readline-like interface for interacting with the build data for a project.",
		&interactiveCmd,
	)
	if err != nil {
		log.Fatal(err)
	}
	interactiveGroup.Aliases = append(interactiveGroup.Aliases, "i")
}

type InteractiveCmd struct{}

var interactiveCmd InteractiveCmd

func helpHandler(l *goline.GoLine) (bool, error) {
	fmt.Println("\nHelp!")
	return false, nil
}

var historyFile = "/tmp/.srclibi_history"

func (c *InteractiveCmd) Execute(args []string) error {
	// Figure out a better way to do this...
	repo, err := OpenRepo(".")
	if err != nil {
		return err
	}
	buildStore, err := buildstore.LocalRepo(repo.RootDir)
	if err != nil {
		return err
	}
	if err := ensureBuild(buildStore, repo); err != nil {
		if err := buildstore.RemoveAllDataForCommit(buildStore, repo.CommitID); err != nil {
			log.Println(err)
		}
		return err
	}

	term, err := terminal(historyFile)
	if err != nil {
		return err
	}
	defer persist(term, historyFile)

	for {
		line, err := term.Prompt("src> ")
		if err != nil {
			if err == io.EOF {
				fmt.Println()
				return nil
			}
			return err
		}
		term.AppendHistory(line)
		words := strings.Fields(line)
		if len(words) == 0 {
			continue
		}
		results, err := parse(words)
		if err != nil {
			return err
		}
		fmt.Println(strings.Join(results, "\n"))
	}
}

type byDefKind struct {
	kind string
}

func (h byDefKind) SelectDef(def *graph.Def) bool {
	return def.Kind == "" || def.Kind == h.kind
}

type defRefs struct {
	def  *graph.Def
	refs []*graph.Ref
}

func parse(inputs []string) (out []string, err error) {
	var queries []string
	var kind string
	for _, input := range inputs {
		if len(input) == 0 {
			continue
		}
		if input[0] == ':' {
			kind = input[1:]
		} else {
			queries = append(queries, input)
		}
	}
	for _, input := range queries {
		c := &StoreDefsCmd{Query: input}
		if kind != "" {
			c.Filter = byDefKind{kind}
		}
		defs, err := c.Get()
		if err != nil {
			return nil, err
		}
		outDefRefs := make([]defRefs, 0, len(defs))
		for _, d := range defs {
			c := &StoreRefsCmd{
				DefRepo:     d.Repo,
				DefUnitType: d.UnitType,
				DefUnit:     d.Unit,
				DefPath:     d.Path,
			}
			refs, err := c.Get()
			if err != nil {
				return nil, err
			}
			outDefRefs = append(outDefRefs, defRefs{d, refs})
		}
		out = append(out, formatObject(outDefRefs))
	}
	return out, nil
}

func getFileSegment(file string, start, end uint32, header bool) string {
	f, err := ioutil.ReadFile(file)
	if err != nil {
		return ""
	}
	if header {
		startLine := bytes.Count(f[:start], []byte{'\n'}) + 1
		// Roll 'start' back and 'end' forward to the nearest
		// newline.
		for ; start-1 > 0 && f[start-1] != '\n'; start-- {
		}
		for ; end < uint32(len(f)) && f[end] != '\n'; end++ {
		}
		var out []string
		onLine := startLine
		for _, line := range bytes.Split(f[start:end], []byte{'\n'}) {
			var marker string
			if startLine == onLine {
				marker = ":"
			} else {
				marker = "-"
			}
			out = append(out, fmt.Sprintf("%s:%d%s%s",
				file, onLine, marker, string(line),
			))
			onLine++
		}
		return strings.Join(out, "\n")
	}
	return string(f[start:end])
}

func formatObject(objs interface{}) string {
	switch o := objs.(type) {
	case *graph.Def:
		if o == nil {
			return "def is nil"
		}
		b, err := json.Marshal(o)
		if err != nil {
			return fmt.Sprintf("error unmarshalling: %s", err)
		}
		c := &FmtCmd{
			UnitType:   o.UnitType,
			ObjectType: "def",
			Format:     "decl",
			Object:     string(b),
		}
		out, err := c.Get()
		if err != nil {
			return fmt.Sprintf("error formatting def: %s", err)
		}
		return fmt.Sprintf("---------- def ----------\n%s\n%s",
			out,
			getFileSegment(o.File, o.DefStart, o.DefEnd, true),
		)
	case []*graph.Def:
		var out []string
		for _, d := range o {
			out = append(out, formatObject(d))
		}
		return strings.Join(out, "\n")
	case *graph.Ref:
		if o == nil {
			return "ref is nil"
		}
		return getFileSegment(o.File, o.Start, o.End, true)
	case []*graph.Ref:
		var out []string
		for _, r := range o {
			if !r.Def {
				out = append(out, formatObject(r))
			}
		}
		return strings.Join(out, "\n")
	case []defRefs:
		var out []string
		for _, d := range o {
			out = append(out, formatObject(d))
		}
		return strings.Join(out, "\n")
	case defRefs:
		var out []string
		out = append(out,
			formatObject(o.def),
			"---------- refs ----------",
			formatObject(o.refs),
		)
		return strings.Join(out, "\n")
	default:
		log.Printf("formatObject: no output for %#v\n", o)
		return ""
	}
}

// from google/cayley
func terminal(path string) (*liner.State, error) {
	term := liner.NewLiner()
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, os.Kill)
		<-c
		err := persist(term, historyFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to properly clean up terminal: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}()
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return term, nil
		}
		return term, err
	}
	defer f.Close()
	_, err = term.ReadHistory(f)
	return term, err
}

// from google/cayley
func persist(term *liner.State, path string) error {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		return fmt.Errorf("could not open %q to append history: %v", path, err)
	}
	defer f.Close()
	_, err = term.WriteHistory(f)
	if err != nil {
		return fmt.Errorf("could not write history to %q: %v", path, err)
	}
	return term.Close()
}
