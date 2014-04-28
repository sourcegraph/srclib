package srcgraph

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"sourcegraph.com/sourcegraph/srcgraph/buildstore"
	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/repo"
	"sourcegraph.com/sourcegraph/srcgraph/scan"
	"sourcegraph.com/sourcegraph/srcgraph/task2"
)

// repository represents a VCS repository on the filesystem. It can be
// autodetected (using detectRepository) or overridden using
// command-line flags (defined by AddRepositoryFlags).
type repository struct {
	CloneURL    string
	CommitID    string
	vcsTypeName string
	RootDir     string
}

func detectRepository(dir string) (dr repository) {
	if !isDir(dir) {
		log.Fatal("dir does not exist: ", dir)
	}

	rootDirCmds := map[string]*exec.Cmd{
		"git": exec.Command("git", "rev-parse", "--show-toplevel"),
		"hg":  exec.Command("hg", "root"),
	}
	for tn, cmd := range rootDirCmds {
		cmd.Dir = dir
		out, err := cmd.Output()
		if err == nil {
			dr.RootDir = strings.TrimSpace(string(out))
			dr.vcsTypeName = tn
			break
		}
	}

	if dr.RootDir == "" {
		if *Verbose {
			log.Printf("warning: failed to detect repository root dir for %q", dir)
		}
		return
	}

	updateVCSIgnore("." + dr.vcsTypeName + "ignore")

	cloneURLCmd := map[string]*exec.Cmd{
		"git": exec.Command("git", "config", "remote.origin.url"),
		"hg":  exec.Command("hg", "paths", "default"),
	}[dr.vcsTypeName]

	vcsType := vcs.VCSByName[dr.vcsTypeName]
	repo, err := vcs.Open(vcsType, dr.RootDir)
	if err != nil {
		if *Verbose {
			log.Printf("warning: failed to open repository at %s: %s", dr.RootDir, err)
		}
		return
	}

	dr.CommitID, err = repo.CurrentCommitID()
	if err != nil {
		return
	}

	cloneURLCmd.Dir = dir
	cloneURL, err := cloneURLCmd.Output()
	if err != nil {
		return
	}
	dr.CloneURL = strings.TrimSpace(string(cloneURL))

	if dr.vcsTypeName == "git" {
		dr.CloneURL = strings.Replace(dr.CloneURL, "git@github.com:", "git://github.com/", 1)
	}

	return
}

func AddRepositoryFlags(fs *flag.FlagSet) *repository {
	r := detectRepository(*dir)
	fs.StringVar(&r.CloneURL, "cloneurl", r.CloneURL, "clone URL of repository")
	fs.StringVar(&r.CommitID, "commit", r.CommitID, "commit ID of current working tree")
	fs.StringVar(&r.vcsTypeName, "vcs", r.vcsTypeName, `VCS type ("git" or "hg")`)
	fs.StringVar(&r.RootDir, "root", r.RootDir, `root directory of repository`)
	return &r
}

func AddRepositoryConfigFlags(fs *flag.FlagSet, r *repository) *repositoryConfigurator {
	rc := &repositoryConfigurator{Repository: r}

	var defaultFile string
	if f, err := findCachedRepoConfigFile(r); err == nil && isFile(f) {
		defaultFile = f
	}

	fs.StringVar(&rc.ConfigFile, "conf.cached", defaultFile, "cached repository config to use (if blank, scans repository for source units and reads .sourcegraph in root dir)")
	fs.BoolVar(&rc.cacheConfig, "conf.cache", true, "cache generated config for repository (saves time on subsequent runs)")
	return rc
}

// findCachedRepoConfigFile determines the filename where the cached
// repository config will be stored, if config caching is enabled.
func findCachedRepoConfigFile(r *repository) (string, error) {
	if r.RootDir == "" {
		return "", fmt.Errorf("no repository root directory")
	}

	repoStore, err := buildstore.NewRepositoryStore(r.RootDir)
	if err != nil {
		return "", err
	}

	rootDataDir, err := buildstore.RootDir(repoStore)
	if err != nil {
		return "", err
	}

	return filepath.Join(rootDataDir, repoStore.CommitPath(r.CommitID), buildstore.CachedRepositoryConfigFilename), nil
}

// repositoryConfigurator gets the *config.Repository for a
// *repository. It uses a cached config if one exists (and cacheConfig
// is true), and otherwise runs the config and scan steps to obtain
// the *config.Repository.
type repositoryConfigurator struct {
	Repository  *repository
	ConfigFile  string
	cacheConfig bool
}

func (rc *repositoryConfigurator) GetRepositoryConfig(x *task2.Context) *config.Repository {
	if rc.ConfigFile != "" {
		f, err := os.Open(rc.ConfigFile)
		if err != nil {
			log.Fatal("error opening repository config file:", err)
		}
		var c *config.Repository
		err = json.NewDecoder(f).Decode(&c)
		if err != nil {
			log.Fatalf("error decoding repository config file %q: %s", rc.ConfigFile, err)
		}
		return c
	}

	c, err := scan.ReadDirConfigAndScan(rc.Repository.RootDir, repo.MakeURI(rc.Repository.CloneURL), x)
	if err != nil {
		log.Fatal(err)
	}

	if rc.cacheConfig {
		configFile, err := findCachedRepoConfigFile(rc.Repository)
		if err != nil {
			log.Fatalf("can't determine filename for repository configuration cache file: %s", err)
		}
		err = os.MkdirAll(filepath.Dir(configFile), 0700)
		if err != nil {
			log.Fatal(err)
		}
		data, err := json.MarshalIndent(c, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		err = ioutil.WriteFile(configFile, data, 0700)
		if err != nil {
			log.Fatal(err)
		}
	}

	return c
}
