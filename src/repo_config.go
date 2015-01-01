package src

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"

	"os"
	"os/exec"
	"strings"
	"syscall"

	"sourcegraph.com/sourcegraph/go-sourcegraph/sourcegraph"
	"sourcegraph.com/sourcegraph/srclib/graph"
)

type Repo struct {
	RootDir  string // Root directory containing repository being analyzed
	VCSType  string // VCS type (git or hg)
	CommitID string // CommitID of current working directory
	CloneURL string // CloneURL of repo.
}

func (c *Repo) URI() string { return graph.MakeURI(c.CloneURL) }

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
	pathCompsInFoundRepo := 0 // find the closest ancestor repo
	for _, vcsType := range []string{"git", "hg"} {
		if d, err := getRootDir(vcsType, dir); err == nil {
			if pathComps := strings.Count(d, string(os.PathSeparator)); pathComps > pathCompsInFoundRepo {
				rc.VCSType = vcsType
				rc.RootDir = d
				pathCompsInFoundRepo = pathComps
			}
		}
	}
	if rc.RootDir == "" {
		return nil, fmt.Errorf("failed to detect git/hg repository root dir for %q; is it in a git/hg repository?", dir)
	}

	var err error
	rc.CommitID, err = resolveWorkingTreeRevision(rc.VCSType, rc.RootDir)
	if err != nil {
		return rc, err
	}

	// Get repo URI from clone URL.
	cloneURL, err := getVCSCloneURL(rc.VCSType, rc.RootDir)
	if err != nil {
		return rc, err
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

// listLatestCommitIDs lists the latest commit ids for dir.
func listLatestCommitIDs(vcsType, dir string) ([]string, error) {
	if vcsType != "git" {
		return nil, fmt.Errorf("listCommitIDs: unsupported vcs type: %q", vcsType)
	}
	cmd := exec.Command("git", "rev-list", "--max-count=5", "HEAD") // 5 picked by random dice roll.
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	return strings.Split(string(bytes.TrimSpace(out)), "\n"), nil
}

// filesChangedToWorkingDir returns a list of the files that have
// changed from fromRev to the current index.
func filesChangedFromRevToIndex(vcsType, dir, fromRev string) ([]string, error) {
	if vcsType != "git" {
		return nil, fmt.Errorf("filesChangedFromRevToHEAD: unsupported vcs type: %q", vcsType)
	}
	cmd := exec.Command("git", "diff", "--name-only", fromRev)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	return strings.Split(string(bytes.TrimSpace(out)), "\n"), nil
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
	rootDir := filepath.Clean(strings.TrimSpace(string(out)))
	return filepath.Abs(rootDir)
}

func getVCSCloneURL(vcsType string, repoDir string) (string, error) {
	run := func(args ...string) (string, error) {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = repoDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			return "", err
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

		url, err = run("git", "config", "remote.origin.url")
		if code, _ := exitStatus(err); code == 1 {
			// `git config --get` returns exit code 1 if the config key doesn't exist.
			return "", errNoVCSCloneURL
		}
		return url, err
	case "hg":
		return run("hg", "--config", "trusted.users=root", "paths", "default")
	default:
		return "", fmt.Errorf("unrecognized VCS %v", vcsType)
	}
}

var errNoVCSCloneURL = errors.New("Could not determine remote clone URL for the current repository. For git repositories, srclib checks for remotes named 'srclib' or 'origin' (in that order). Run 'git remote add NAME URL' to add a remote, where NAME is either 'srclib' or 'origin' and URL is a git clone URL (e.g. https://example.com/repo.git).' to add a remote. For hg repositories, srclib checks the 'default' remote.")

func exitStatus(err error) (uint32, error) {
	if err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			// There is no platform independent way to retrieve
			// the exit code, but the following will work on Unix
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				return uint32(status.ExitStatus()), nil
			}
		}
		return 0, err
	}
	return 0, nil
}
