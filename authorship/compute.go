package authorship

import (
	"log"
	"sort"
	"time"

	"github.com/sourcegraph/go-nnz/nnz"

	"sourcegraph.com/sourcegraph/srclib/graph"
	"sourcegraph.com/sourcegraph/srclib/grapher"
	"sourcegraph.com/sourcegraph/srclib/repo"
	"sourcegraph.com/sourcegraph/srclib/vcsutil"
)

func ComputeSourceUnit(g *grapher.Output, b *vcsutil.BlameOutput) (*SourceUnitOutput, error) {
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
	o.Defs = make(map[graph.DefPath][]*DefAuthorship, len(g.Defs))

	for _, def := range g.Defs {
		authors, chars, err := getAuthors(def.File, def.DefStart, def.DefEnd)
		if err != nil {
			return nil, err
		}
		totalDefDefChars := float64(def.DefEnd - def.DefStart)
		for _, author := range authors {
			charsProportion := float64(0.0)
			if totalDefDefChars != 0 {
				charsProportion = float64(chars[author.AuthorEmail]) / totalDefDefChars
			}
			o.Defs[def.Path] = append(o.Defs[def.Path], &DefAuthorship{
				AuthorshipInfo:  *author,
				Exported:        def.Exported,
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

	var totalDefs, totalExportedDefs int
	authorsByEmail := make(map[string]*AuthorStats)
	for _, sas := range o.Defs {
		totalDefs++
		if sas[0].Exported {
			totalExportedDefs++
		}

		for _, sa := range sas {
			ra, present := authorsByEmail[sa.AuthorEmail]
			if !present {
				ra = new(AuthorStats)
				ra.AuthorEmail = sa.AuthorEmail
				authorsByEmail[sa.AuthorEmail] = ra
			}

			ra.DefCount++
			if sa.Exported {
				ra.ExportedDefCount++
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
		if totalDefs != 0 {
			ra.DefsProportion = float64(ra.DefCount) / float64(totalDefs)
		}
		if totalExportedDefs != 0 {
			ra.ExportedDefsProportion = float64(ra.ExportedDefCount) / float64(totalExportedDefs)
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

		key := clientKey{ra.DefRepo, ra.DefUnitType, ra.DefUnit}
		rc, present := clientMap[key]
		if !present {
			rc = &ClientStats{}
			rc.AuthorEmail = ra.AuthorEmail
			rc.DefRepo = ra.DefRepo
			rc.DefUnitType = nnz.String(ra.DefUnitType)
			rc.DefUnit = nnz.String(ra.DefUnit)
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
