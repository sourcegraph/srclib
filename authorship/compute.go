package authorship

import (
	"log"
	"sort"
	"time"

	"github.com/sourcegraph/go-nnz/nnz"

	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/graph"
	"sourcegraph.com/sourcegraph/srcgraph/grapher2"
	"sourcegraph.com/sourcegraph/srcgraph/repo"
	"sourcegraph.com/sourcegraph/srcgraph/vcsutil"
)

func ComputeSourceUnit(g *grapher2.Output, b *vcsutil.BlameOutput, c *config.Repository) (*SourceUnitOutput, error) {
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
					log.Printf("warning: no commit ID %q for hunk %+v in file %s", h.CommitID, h, file)
				}

				nchars := min(end, h.CharEnd) - max(start, h.CharStart)
				if a, present := authorsByEmail[commit.Author.Email]; present {
					// user contributed to 2+ hunks for this
					if a.LastCommitDate.Before(commit.AuthorDate) {
						// take most recent author date
						a.LastCommitDate = commit.AuthorDate.In(time.UTC)
						a.LastCommitID = commit.ID
					}
					charsByEmail[commit.Author.Email] += nchars
				} else {
					authorsByEmail[commit.Author.Email] = &AuthorshipInfo{
						AuthorEmail:    commit.Author.Email,
						LastCommitDate: commit.AuthorDate.In(time.UTC),
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
			charsProportion := float64(0.0)
			if totalSymbolDefChars != 0 {
				charsProportion = float64(chars[author.AuthorEmail]) / totalSymbolDefChars
			}
			o.Symbols[sym.Path] = append(o.Symbols[sym.Path], &SymbolAuthorship{
				AuthorshipInfo:  *author,
				Exported:        sym.Exported,
				Chars:           chars[author.AuthorEmail],
				CharsProportion: charsProportion,
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

	var totalSymbols, totalExportedSymbols int
	authorsByEmail := make(map[string]*AuthorStats)
	for _, sas := range o.Symbols {
		totalSymbols++
		if sas[0].Exported {
			totalExportedSymbols++
		}

		for _, sa := range sas {
			ra, present := authorsByEmail[sa.AuthorEmail]
			if !present {
				ra = new(AuthorStats)
				ra.AuthorEmail = sa.AuthorEmail
				authorsByEmail[sa.AuthorEmail] = ra
			}

			ra.SymbolCount++
			if sa.Exported {
				ra.ExportedSymbolCount++
			}

			if ra.LastCommitDate.Before(sa.LastCommitDate) {
				ra.LastCommitDate = sa.LastCommitDate.In(time.UTC)
				ra.LastCommitID = sa.LastCommitID
			}
		}
	}
	// calculate proportions
	o.Authors = make([]*AuthorStats, len(authorsByEmail))
	i := 0
	for _, ra := range authorsByEmail {
		if totalSymbols != 0 {
			ra.SymbolsProportion = float64(ra.SymbolCount) / float64(totalSymbols)
		}
		if totalExportedSymbols != 0 {
			ra.ExportedSymbolsProportion = float64(ra.ExportedSymbolCount) / float64(totalExportedSymbols)
		}

		o.Authors[i] = ra
		i++
	}

	type clientKey struct {
		Repo     repo.URI
		UnitType string
		Unit     string
	}
	clientsByEmail := make(map[string]map[clientKey]*ClientStats)
	for _, ra := range o.Refs {
		clientMap, present := clientsByEmail[ra.AuthorEmail]
		if !present {
			clientMap = make(map[clientKey]*ClientStats)
			clientsByEmail[ra.AuthorEmail] = clientMap
		}

		key := clientKey{ra.SymbolRepo, ra.SymbolUnitType, ra.SymbolUnit}
		rc, present := clientMap[key]
		if !present {
			rc = &ClientStats{}
			rc.AuthorEmail = ra.AuthorEmail
			rc.SymbolRepo = ra.SymbolRepo
			rc.SymbolUnitType = nnz.String(ra.SymbolUnitType)
			rc.SymbolUnit = nnz.String(ra.SymbolUnit)
			clientMap[key] = rc
		}

		if rc.LastCommitDate.Before(ra.LastCommitDate) {
			rc.LastCommitDate = ra.LastCommitDate.In(time.UTC)
			rc.LastCommitID = ra.LastCommitID
		}

		rc.RefCount++
	}
	// convert to array
	for _, clientMap := range clientsByEmail {
		for _, rc := range clientMap {
			o.ClientsOfOtherUnits = append(o.ClientsOfOtherUnits, rc)
		}
	}

	return sortedOutput(&o), nil
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

func sortedOutput(o *SourceUnitOutput) *SourceUnitOutput {
	sort.Sort(refAuthorships(o.Refs))
	sort.Sort(authorStats(o.Authors))
	sort.Sort(clientsOfOtherUnits(o.ClientsOfOtherUnits))
	return o
}

type refAuthorships []*RefAuthorship

func (p refAuthorships) Len() int           { return len(p) }
func (p refAuthorships) Less(i, j int) bool { return p[i].sortKey() < p[j].sortKey() }
func (p refAuthorships) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

type authorStats []*AuthorStats

func (p authorStats) Len() int           { return len(p) }
func (p authorStats) Less(i, j int) bool { return p[i].sortKey() < p[j].sortKey() }
func (p authorStats) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

type clientsOfOtherUnits []*ClientStats

func (p clientsOfOtherUnits) Len() int           { return len(p) }
func (p clientsOfOtherUnits) Less(i, j int) bool { return p[i].sortKey() < p[j].sortKey() }
func (p clientsOfOtherUnits) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
