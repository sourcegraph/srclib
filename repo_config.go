package srcgraph

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sourcegraph/go-vcs/vcs"
	"sourcegraph.com/sourcegraph/srcgraph/buildstore"
	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/repo"
	"sourcegraph.com/sourcegraph/srcgraph/scan"
)

type RepoContext struct {
	RepoRootDir string // Root directory containing repository being analyzed
	VCSType     string // VCS type (git or hg)
	CommitID    string // CommitID of current working directory
}

func NewRepoContext(targetDir string) (*RepoContext, error) {
	if !isDir(targetDir) {
		return nil, fmt.Errorf("directory not exist: %q", targetDir)
	}

	// VCS and root directory
	rc := new(RepoContext)
	for _, vcsType := range []string{"git", "hg"} {
		if d, err := getRepoRootDir(vcsType, targetDir); err == nil {
			rc.VCSType = vcsType
			rc.RepoRootDir = d
			break
		}
	}
	if rc.RepoRootDir == "" {
		return nil, fmt.Errorf("warning: failed to detect repository root dir for %q", targetDir)
	}

	// Determine current working tree commit ID.
	repo, err := vcs.Open(rc.VCSType, rc.RepoRootDir)
	if err != nil {
		return nil, err
	}
	var currentRevSpec string
	switch rc.VCSType {
	case "git":
		currentRevSpec = "HEAD"
	case "hg":
		currentRevSpec = "tip"
	}
	currentCommitID, err := repo.ResolveRevision(currentRevSpec)
	if err != nil {
		return nil, err
	}

	rc.CommitID = string(currentCommitID)

	// updateVCSIgnore("." + rc.VCS.ShortName() + "ignore") // TODO: desirable?
	return rc, nil
}

type JobContext struct {
	*RepoContext
	Repo *config.Repository // Repository-level configuration, read from .sourcegraph-data/config.json
}

func NewJobContext(targetDir string) (*JobContext, error) {
	rc, err := NewRepoContext(targetDir)
	if err != nil {
		return nil, err
	}
	jc := new(JobContext)
	jc.RepoContext = rc

	// get default URI (if URI is not specified in .sourcegraph file)
	// TODO(bliu): this seems like it should get pushed into ReadOrComputeRepositoryConfig...
	cloneURL, err := getVCSCloneURL(rc.VCSType, targetDir)
	if err != nil {
		return nil, err
	}
	uri := repo.MakeURI(cloneURL)

	jc.Repo, err = ReadOrComputeRepositoryConfig(jc.RepoRootDir, jc.CommitID, uri)
	if err != nil {
		return nil, err
	}

	return jc, nil
}

func ReadOrComputeRepositoryConfig(repoDir string, commitID string, repoURI repo.URI) (*config.Repository, error) {
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
		return scan.ReadDirConfigAndScan(repoDir, repoURI)
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
	if err != nil && !os.IsExist(err) {
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

func getRepoRootDir(vcsType string, dir string) (string, error) {
	var cmd *exec.Cmd
	switch vcsType {
	case "git":
		cmd = exec.Command("git", "rev-parse", "--show-toplevel")
	case "hg":
		cmd = exec.Command("hg", "root")
	}
	if cmd == nil {
		return "", fmt.Errorf("unrecognized VCS %v", vcsType)
	}
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func getVCSCloneURL(vcsType string, repoDir string) (string, error) {
	var cmd *exec.Cmd
	switch vcsType {
	case "git":
		cmd = exec.Command("git", "config", "remote.origin.url")
	case "hg":
		cmd = exec.Command("hg", "paths", "default")
	}
	if cmd == nil {
		return "", fmt.Errorf("unrecognized VCS %v", vcsType)
	}
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("could not get VCS URL: %s", err)
	}

	cloneURL := strings.TrimSpace(string(out))
	if vcsType == "git" {
		cloneURL = strings.Replace(cloneURL, "git@github.com:", "git://github.com/", 1)
	}
	return cloneURL, nil
}
