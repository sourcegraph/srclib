package src

import (
	"bytes"
	"fmt"

	"os"
	"os/exec"
	"strings"

	"sourcegraph.com/sourcegraph/go-sourcegraph/sourcegraph"
	"sourcegraph.com/sourcegraph/srclib/graph"
)

type Repo struct {
	RootDir  string // Root directory containing repository being analyzed
	VCSType  string // VCS type (git or hg)
	CommitID string // CommitID of current working directory
	CloneURL string // CloneURL of repo.
}

func (c *Repo) URI() string {
	uri := graph.MakeURI(c.CloneURL)
	// TODO(sqs): temp workaround for sourcegraph private repo
	if uri == "github.com/sourcegraph/sourcegraph" {
		return "sourcegraph.com/sourcegraph/sourcegraph"
	}
	return uri
}

func (r *Repo) RepoRevSpec() sourcegraph.RepoRevSpec {
	return sourcegraph.RepoRevSpec{
		RepoSpec: sourcegraph.RepoSpec{URI: r.URI()},
		Rev:      r.CommitID,
		CommitID: r.CommitID,
	}
}

func OpenRepo(dir string) (*Repo, error) {
	if fi, err := os.Stat(dir); err != nil || !fi.Mode().IsDir() {
		return nil, fmt.Errorf("not a directory: %q", dir)
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

	var err error
	rc.CommitID, err = resolveWorkingTreeRevision(rc.VCSType, rc.RootDir)
	if err != nil {
		return nil, err
	}

	// Get repo URI from clone URL.
	cloneURL, err := getVCSCloneURL(rc.VCSType, rc.RootDir)
	if err != nil {
		return nil, err
	}
	rc.CloneURL = cloneURL

	updateVCSIgnore("." + rc.VCSType + "ignore")
	return rc, nil
}

func resolveWorkingTreeRevision(vcsType string, dir string) (string, error) {
	var cmd *exec.Cmd
	switch vcsType {
	case "git":
		cmd = exec.Command("git", "rev-parse", "HEAD")
	case "hg":
		cmd = exec.Command("hg", "--config", "trusted.users=root", "identify", "--debug", "-i")
	default:
		return "", fmt.Errorf("unknown vcs type: %q", vcsType)
	}
	cmd.Dir = dir

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("exec %v failed: %s. Output was:\n\n%s", cmd.Args, err, out)
	}
	// hg adds a "+" if the wd is dirty
	return strings.TrimSuffix(string(bytes.TrimSpace(out)), "+"), nil
}

func getRootDir(vcsType string, dir string) (string, error) {
	var cmd *exec.Cmd
	switch vcsType {
	case "git":
		cmd = exec.Command("git", "rev-parse", "--show-toplevel")
	case "hg":
		cmd = exec.Command("hg", "--config", "trusted.users=root", "root")
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
	run := func(args ...string) (string, error) {
		cmd := exec.Command(args[0], args[1:]...)
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
	switch vcsType {
	case "git":
		// Try to get the "srclib" remote first.
		url, err := run("git", "config", "remote.srclib.url")
		if err == nil {
			return url, nil
		}

		return run("git", "config", "remote.origin.url")
	case "hg":
		return run("hg", "--config", "trusted.users=root", "paths", "default")
	default:
		return "", fmt.Errorf("unrecognized VCS %v", vcsType)
	}
}
