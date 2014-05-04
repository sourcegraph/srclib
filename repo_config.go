package srcgraph

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sourcegraph/go-vcs"

	"sourcegraph.com/sourcegraph/srcgraph/buildstore"
	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/repo"
	"sourcegraph.com/sourcegraph/srcgraph/scan"
	"sourcegraph.com/sourcegraph/srcgraph/task2"
)

type RepoContext struct {
	RepoRootDir string  // Root directory containing repository being analyzed
	VCS         vcs.VCS // VCS type (git or hg)
	CommitID    string  // CommitID of current working directory
}

func NewRepoContext(targetDir string) (*RepoContext, error) {
	if !isDir(targetDir) {
		return nil, fmt.Errorf("directory not exist: %q", targetDir)
	}

	// VCS and root directory
	rc := new(RepoContext)
	for _, v := range vcs.VCSByName {
		if d, err := getRepoRootDir(v, targetDir); err == nil {
			rc.VCS = v
			rc.RepoRootDir = d
			break
		}
	}
	if rc.RepoRootDir == "" {
		return nil, fmt.Errorf("warning: failed to detect repository root dir for %q", targetDir)
	}

	// CommitID
	repo, err := vcs.Open(rc.VCS, rc.RepoRootDir)
	if err != nil {
		return nil, err
	}
	rc.CommitID, err = repo.CurrentCommitID()
	if err != nil {
		return nil, err
	}

	// updateVCSIgnore("." + rc.VCS.ShortName() + "ignore") // TODO: desirable?
	return rc, nil
}

type JobContext struct {
	*RepoContext
	Repo *config.Repository // Repository-level configuration, read from .sourcegraph-data/config.json
}

func NewJobContext(targetDir string, x *task2.Context) (*JobContext, error) {
	rc, err := NewRepoContext(targetDir)
	if err != nil {
		return nil, err
	}
	jc := new(JobContext)
	jc.RepoContext = rc

	// get default URI (if URI is not specified in .sourcegraph file)
	// TODO(bliu): this seems like it should get pushed into ReadOrComputeRepositoryConfig...
	cloneURL, err := getVCSCloneURL(rc.VCS, targetDir)
	if err != nil {
		return nil, err
	}
	uri := repo.MakeURI(cloneURL)

	jc.Repo, err = ReadOrComputeRepositoryConfig(jc.RepoRootDir, jc.CommitID, uri, x)
	if err != nil {
		return nil, err
	}

	return jc, nil
}

func ReadOrComputeRepositoryConfig(repoDir string, commitID string, repoURI repo.URI, x *task2.Context) (*config.Repository, error) {
	configFile, err := getConfigFile(repoDir, commitID)
	if err != nil {
		return nil, err
	}
	if isFile(configFile) {
		// Read
		f, err := os.Open(configFile)
		if err != nil {
			return nil, err
		}
		var c config.Repository
		err = json.NewDecoder(f).Decode(&c)
		if err != nil {
			return nil, err
		}
		return &c, nil
	} else {
		// Compute
		return scan.ReadDirConfigAndScan(repoDir, repoURI, x)
	}
}

func WriteRepositoryConfig(repoDir string, commitID string, c *config.Repository, overwrite bool) error {
	configFile, err := getConfigFile(repoDir, commitID)
	if err != nil {
		return err
	}
	if isFile(configFile) && !overwrite {
		return nil
	}

	err = os.MkdirAll(filepath.Dir(configFile), 0700)
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(configFile, b, 0700)
}

func getConfigFile(repoDir, commitID string) (string, error) {
	if repoDir == "" {
		return "", fmt.Errorf("no repository root directory")
	}
	repoStore, err := buildstore.NewRepositoryStore(repoDir)
	if err != nil {
		return "", err
	}
	rootDataDir, err := buildstore.RootDir(repoStore)
	if err != nil {
		return "", err
	}
	return filepath.Join(rootDataDir, repoStore.CommitPath(commitID), buildstore.CachedRepositoryConfigFilename), nil
}

func getRepoRootDir(v vcs.VCS, dir string) (string, error) {
	var cmd *exec.Cmd
	switch v {
	case vcs.Git:
		cmd = exec.Command("git", "rev-parse", "--show-toplevel")
	case vcs.Hg:
		cmd = exec.Command("hg", "root")
	}
	if cmd == nil {
		return "", fmt.Errorf("unrecognized VCS %v", v)
	}
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func getVCSCloneURL(v vcs.VCS, repoDir string) (string, error) {
	var cmd *exec.Cmd
	switch v {
	case vcs.Git:
		cmd = exec.Command("git", "config", "remote.origin.url")
	case vcs.Hg:
		cmd = exec.Command("hg", "paths", "default")
	}
	if cmd == nil {
		return "", fmt.Errorf("unrecognized VCS %v", v)
	}
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	cloneURL := strings.TrimSpace(string(out))
	if v == vcs.Git {
		cloneURL = strings.Replace(cloneURL, "git@github.com", "git://github.com/", 1)
	}
	return cloneURL, nil
}
