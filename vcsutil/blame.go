package vcsutil

import (
	"os"
	"path/filepath"
	"time"

	"github.com/sourcegraph/go-blame/blame"
	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/task2"
	"sourcegraph.com/sourcegraph/util"
)

var SkipBlame = util.ParseBool(os.Getenv("SG_SKIP_BLAME"))

type BlameOutput struct {
	CommitMap map[string]blame.Commit
	HunkMap   map[string][]blame.Hunk
}

var blameIgnores = []string{
	"node_modules", "bower_components",
	"doc", "docs", "build", "vendor",
	".min.js", "-min.js", ".optimized.js", "-optimized.js",
	"dist", "assets", "deps/", "dep/", ".jar", ".png", ".html",
	"third-party",
}

func BlameRepository(dir string, commitID string, c *config.Repository, x *task2.Context) (*BlameOutput, error) {
	blameOutput := new(BlameOutput)
	if SkipBlame {
		x.Log.Printf("Skipping VCS blame (returning empty BlameOutput)")
		return blameOutput, nil
	}

	var err error
	blameOutput.HunkMap, blameOutput.CommitMap, err = blame.BlameRepository(dir, commitID, nil)
	return utcTime(blameOutput), err
}

func BlameFiles(dir string, files []string, commitID string, c *config.Repository, x *task2.Context) (*BlameOutput, error) {
	if SkipBlame {
		x.Log.Printf("Skipping VCS blame (returning empty BlameOutput)")
		return new(BlameOutput), nil
	}

	hunkMap := make(map[string][]blame.Hunk)
	commitMap := make(map[string]blame.Commit)

	for _, file := range files {
		relFile, err := filepath.Rel(dir, file)
		if err != nil {
			return nil, err
		}

		hunks, commitMap2, err := blame.BlameFile(dir, relFile, commitID)
		if err != nil {
			return nil, err
		}
		hunkMap[relFile] = hunks
		for cid, cm := range commitMap2 {
			if _, present := commitMap[cid]; !present {
				commitMap[cid] = cm
			}
		}
	}

	return utcTime(&BlameOutput{commitMap, hunkMap}), nil
}

// utcTime sets the commit timestamps to UTC. PERF TODO(sqs): This is very
// inefficient because the map values are not pointers.
func utcTime(o *BlameOutput) *BlameOutput {
	for id, c := range o.CommitMap {
		c.AuthorDate = c.AuthorDate.In(time.UTC)
		o.CommitMap[id] = c
	}
	return o
}
