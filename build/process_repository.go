package build

import (
	"log"
	"os"
	"path/filepath"

	"github.com/sourcegraph/go-vcs"
	"sourcegraph.com/sourcegraph/repo"
	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/grapher2"
	"sourcegraph.com/sourcegraph/srcgraph/scan"
	"sourcegraph.com/sourcegraph/srcgraph/task2"
	_ "sourcegraph.com/sourcegraph/srcgraph/toolchain/all_toolchains"
)

var (
	WorkDir = filepath.Join(os.TempDir(), "sg")
	DryRun  bool
)

func init() {
	err := os.MkdirAll(WorkDir, 0700)
	if err != nil {
		log.Fatal(err)
	}
}

type RepositoryData struct {
	Config *config.Repository

	CommitID string

	// TODO(sqs): add RefAuthors, SymbolAuthors (derived from VCS hunk data)

	Graph map[string]*grapher2.Output
}

// Repository creates a plan to fully process a specific revision of a
// repository.
func Repository(dir string, commitID string, cloneURL string, vcsType vcs.VCS, x *task2.Context) ([]task2.Task, *RepositoryData, error) {
	repoURI := repo.MakeURI(cloneURL)
	c, err := scan.ReadDirConfigAndScan(dir, repoURI, x)
	if err != nil {
		return nil, nil, err
	}

	rp := &repositoryPlanner{dir, commitID, x, c, nil}
	return rp.planTasks()
}
