package src

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/kr/fs"

	"sourcegraph.com/sourcegraph/go-sourcegraph/sourcegraph"
	"sourcegraph.com/sourcegraph/srclib/buildstore"
	"sourcegraph.com/sourcegraph/srclib/dep"
	"sourcegraph.com/sourcegraph/srclib/graph"
)

func init() {
	queryGroup, err := CLI.AddCommand("query",
		"search code in current project and dependencies",
		"The query (q) command searches for code in the current project and its dependencies. The results include documentation, definitions, etc.",
		&queryCmd,
	)
	if err != nil {
		log.Fatal(err)
	}
	queryGroup.Aliases = append(queryGroup.Aliases, "q")
}

type QueryCmd struct{}

var queryCmd QueryCmd

func (c *QueryCmd) Execute(args []string) error {
	repo, err := OpenRepo(".")
	if err != nil {
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

	cl := NewAPIClientWithAuthIfPresent()

	repoAndDepURIs := []string{repo.URI()}

	// Read deps.
	depSuffix := buildstore.DataTypeSuffix([]*dep.ResolvedDep{})
	w := fs.WalkFS(".", commitFS)
	seenDepURI := map[string]bool{}
	for w.Step() {
		depfile := w.Path()
		if strings.HasSuffix(depfile, depSuffix) {
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
				if d.Target != nil && d.Target.ToRepoCloneURL != "" {
					depURI := graph.MakeURI(d.Target.ToRepoCloneURL)
					if !seenDepURI[depURI] {
						repoAndDepURIs = append(repoAndDepURIs, depURI)
						seenDepURI[depURI] = true
					}
				}
			}
		}
	}

	for _, repoURI := range repoAndDepURIs {
		query := fmt.Sprintf("repo:%s %s", repoURI, strings.Join(args, " "))
		if GlobalOpt.Verbose {
			log.Printf("# Query: %q", query)
		}
		res, _, err := cl.Search.Search(&sourcegraph.SearchOptions{
			Query:       query,
			Defs:        true,
			ListOptions: sourcegraph.ListOptions{PerPage: 5},
		})
		if err != nil {
			if GlobalOpt.Verbose {
				log.Println(err)
			}
			continue
		}
		defs := res.Defs

		for _, def := range defs {
			// Fetch docs and stats.
			def, _, err = cl.Defs.Get(def.DefSpec(), &sourcegraph.DefGetOptions{Doc: true})
			if err != nil {
				return err
			}

			if f := def.FmtStrings; f != nil {
				fromDep := !graph.URIEqual(def.Repo, repo.URI())

				kw := f.DefKeyword
				if kw != "" {
					kw += " "
				}

				var name string
				if fromDep {
					name = f.Name.LanguageWideQualified
				} else {
					name = f.Name.DepQualified
				}

				var typ string
				if fromDep {
					typ = f.Type.RepositoryWideQualified
				} else {
					typ = f.Type.DepQualified
				}

				fmt.Printf("%s%s%s%s\n", kw, bold(red(name)), f.NameAndTypeSeparator, bold(typ))
			} else {
				fmt.Printf("(unable to format: %s from %s)\n", def.Name, def.Repo)
			}

			if doc := strings.TrimSpace(stripHTML(def.DocHTML)); doc != "" {
				fmt.Println(doc, "    ")
			}

			src := fmt.Sprintf("@ %s", def.File)

			var stat string
			if def.RRefs() > 0 || def.XRefs() > 0 {
				stat = fmt.Sprintf("%d xrefs %d rrefs", def.XRefs(), def.RRefs())
			}

			fmt.Printf("%-50s %s\n", fade(src), fade(stat))

			fmt.Println()
		}
	}
	return nil
}

func stripHTML(html string) string {
	return strings.Replace(strings.Replace(html, "<p>", "", -1), "</p>", "", -1)
}
