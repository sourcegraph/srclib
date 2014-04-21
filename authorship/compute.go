package authorship

import (
	"fmt"
	"log"

	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/graph"
	"sourcegraph.com/sourcegraph/srcgraph/grapher2"
	"sourcegraph.com/sourcegraph/srcgraph/repo"
	"sourcegraph.com/sourcegraph/srcgraph/task2"
	"sourcegraph.com/sourcegraph/srcgraph/vcsutil"
)

func ComputeSourceUnit(g *grapher2.Output, b *vcsutil.BlameOutput, c *config.Repository, x *task2.Context) (*SourceUnitOutput, error) {
	var getAuthors = func(file string, start, end int) (map[string]*AuthorshipInfo, map[string]int, error) {
		hunks, present := b.HunkMap[file]
		if !present {
			log.Printf("No hunk for file %q", file)
			return nil, nil, nil
		}

		authorsByEmail := make(map[string]*AuthorshipInfo)
		charsByEmail := make(map[string]int)
		for _, h := range hunks {
			if h.CharStart <= end && h.CharEnd > start {
				commit, present := b.CommitMap[h.CommitID]
				if !present {
					return nil, nil, fmt.Errorf("no commit ID %q for hunk %+v", h.CommitID, h)
				}

				nchars := min(end, h.CharEnd) - max(start, h.CharStart)
				if a, present := authorsByEmail[commit.Author.Email]; present {
					// user contributed to 2+ hunks for this
					if a.LastCommitDate.Before(commit.AuthorDate) {
						// take most recent author date
						a.LastCommitDate = commit.AuthorDate
						a.LastCommitID = commit.ID
					}
					charsByEmail[commit.Author.Email] += nchars
				} else {
					authorsByEmail[commit.Author.Email] = &AuthorshipInfo{
						AuthorEmail:    commit.Author.Email,
						LastCommitDate: commit.AuthorDate,
						LastCommitID:   commit.ID,
					}
					charsByEmail[commit.Author.Email] = nchars
				}
			}
		}
		return authorsByEmail, charsByEmail, nil
	}

	var o SourceUnitOutput
	o.Symbols = make(map[graph.SymbolPath][]*SymbolAuthorship, len(g.Symbols))

	for _, sym := range g.Symbols {
		authors, chars, err := getAuthors(sym.File, sym.DefStart, sym.DefEnd)
		if err != nil {
			return nil, err
		}
		totalSymbolDefChars := float64(sym.DefEnd - sym.DefStart)
		for _, author := range authors {
			o.Symbols[sym.Path] = append(o.Symbols[sym.Path], &SymbolAuthorship{
				AuthorshipInfo:  *author,
				Exported:        sym.Exported,
				Chars:           chars[author.AuthorEmail],
				CharsProportion: float64(chars[author.AuthorEmail]) / totalSymbolDefChars,
			})
		}
	}

	for _, ref := range g.Refs {
		authors, _, err := getAuthors(ref.File, ref.Start, ref.End)
		if err != nil {
			return nil, err
		}
		for _, author := range authors {
			o.Refs = append(o.Refs, &RefAuthorship{
				AuthorshipInfo: *author,
				RefKey:         ref.RefKey(),
			})
		}
	}

	return &o, nil
}

func ComputeRepository(unitOutputs []*SourceUnitOutput, c *config.Repository, x *task2.Context) (*RepositoryOutput, error) {
	var o RepositoryOutput

	var totalSymbols, totalExportedSymbols int

	authorsByEmail := make(map[string]*RepositoryAuthorship)
	clientsByEmail := make(map[string]map[repo.URI]*RepositoryClientship)
	for _, uo := range unitOutputs {
		for _, sas := range uo.Symbols {
			totalSymbols++
			if sas[0].Exported {
				totalExportedSymbols++
			}

			for _, sa := range sas {
				ra, present := authorsByEmail[sa.AuthorEmail]
				if !present {
					ra = new(RepositoryAuthorship)
					ra.AuthorEmail = sa.AuthorEmail
					authorsByEmail[sa.AuthorEmail] = ra
				}

				ra.SymbolCount++
				if sa.Exported {
					ra.ExportedSymbolCount++
				}

				if ra.LastCommitDate.Before(sa.LastCommitDate) {
					ra.LastCommitDate = sa.LastCommitDate
					ra.LastCommitID = sa.LastCommitID
				}
			}
		}

		for _, ra := range uo.Refs {
			clientMap, present := clientsByEmail[ra.AuthorEmail]
			if !present {
				clientMap = make(map[repo.URI]*RepositoryClientship)
				clientsByEmail[ra.AuthorEmail] = clientMap
			}

			rc, present := clientMap[ra.SymbolRepo]
			if !present {
				rc = new(RepositoryClientship)
				rc.AuthorEmail = ra.AuthorEmail
				rc.SymbolRepo = ra.SymbolRepo
				clientMap[ra.SymbolRepo] = rc
			}

			if rc.LastCommitDate.Before(ra.LastCommitDate) {
				rc.LastCommitDate = ra.LastCommitDate
				rc.LastCommitID = ra.LastCommitID
			}

			rc.RefCount++
		}
	}

	// calculate proportions
	o.Authors = make([]*RepositoryAuthorship, len(authorsByEmail))
	i := 0
	for _, ra := range authorsByEmail {
		ra.SymbolsProportion = float64(ra.SymbolCount) / float64(totalSymbols)
		ra.ExportedSymbolsProportion = float64(ra.ExportedSymbolCount) / float64(totalExportedSymbols)

		o.Authors[i] = ra
		i++
	}

	// convert to array
	for _, clientMap := range clientsByEmail {
		for _, rc := range clientMap {
			o.ClientsOfOtherRepositories = append(o.ClientsOfOtherRepositories, rc)
		}
	}

	return &o, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
