package src

import (
	"fmt"

	"os/exec"
	"strings"

	"github.com/sourcegraph/go-vcs/vcs"
	"github.com/sourcegraph/srclib/repo"
)

type Repo struct {
	RootDir  string // Root directory containing repository being analyzed
	VCSType  string // VCS type (git or hg)
	CommitID string // CommitID of current working directory
	CloneURL string // CloneURL of repo.
}

func (c *Repo) URI() repo.URI { return repo.MakeURI(c.CloneURL) }

func OpenRepo(dir string) (*Repo, error) {
	if !isDir(dir) {
		return nil, fmt.Errorf("no such directory: %q", dir)
	}

	// VCS and root directory
	rc := new(Repo)
	for _, vcsType := range []string{"git", "hg"} {
		if d, err := getRootDir(vcsType, dir); err == nil {
			rc.VCSType = vcsType
			rc.RootDir = d
			break
		}
	}
	if rc.RootDir == "" {
		return nil, fmt.Errorf("failed to detect repository root dir for %q", dir)
	}

	// Determine current working tree commit ID.
	repo, err := vcs.Open(rc.VCSType, rc.RootDir)
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

	// Get repo URI from clone URL.
	cloneURL, err := getVCSCloneURL(rc.VCSType, rc.RootDir)
	if err != nil {
		return nil, err
	}
	rc.CloneURL = cloneURL

	updateVCSIgnore("." + rc.VCSType + "ignore")
	return rc, nil
}

func getRootDir(vcsType string, dir string) (string, error) {
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
