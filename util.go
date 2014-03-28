package srcgraph

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/sourcegraph/go-vcs"
	"sourcegraph.com/sourcegraph/srcgraph/buildstore"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
)

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

func AddRepositoryFlags(fs *flag.FlagSet) repository {
	dr := detectRepository(*dir)
	var r repository
	fs.StringVar(&r.CloneURL, "cloneurl", dr.CloneURL, "clone URL of repository")
	fs.StringVar(&r.CommitID, "commit", dr.CommitID, "commit ID of current working tree")
	fs.StringVar(&r.vcsTypeName, "vcs", dr.vcsTypeName, `VCS type ("git" or "hg")`)
	fs.StringVar(&r.RootDir, "root", dr.RootDir, `root directory of repository`)
	return r
}

func isDir(dir string) bool {
	di, err := os.Stat(dir)
	return err == nil && di.IsDir()
}

func isFile(file string) bool {
	fi, err := os.Stat(file)
	return err == nil && fi.Mode().IsRegular()
}

func firstLine(s string) string {
	i := strings.Index(s, "\n")
	if i == -1 {
		return s
	}
	return s[:i]
}

func cmdOutput(c ...string) string {
	cmd := exec.Command(c[0], c[1:]...)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		log.Fatalf("%v: %s", c, err)
	}
	return strings.TrimSpace(string(out))
}

func SourceUnitMatchesArgs(specified []string, u unit.SourceUnit) bool {
	var match bool
	if len(specified) == 0 {
		match = true
	} else {
		for _, unitSpec := range specified {
			if string(unit.MakeID(u)) == unitSpec || u.Name() == unitSpec {
				match = true
				break
			}
		}
	}

	return match
}

func PrintJSON(v interface{}, prefix string) {
	data, err := json.MarshalIndent(v, prefix, "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}

func OpenInputFiles(extraArgs []string) map[string]io.ReadCloser {
	inputs := make(map[string]io.ReadCloser)
	if len(extraArgs) == 0 {
		inputs["<stdin>"] = os.Stdin
	} else {
		for _, name := range extraArgs {
			f, err := os.Open(name)
			if err != nil {
				log.Fatal(err)
			}
			inputs[name] = f
		}
	}
	return inputs
}

func CloseAll(files map[string]io.ReadCloser) {
	for _, rc := range files {
		rc.Close()
	}
}

// updateVCSIgnore adds .sourcegraph-data/ to the user's .${VCS}ignore file in
// their home directory.
func updateVCSIgnore(name string) {
	u, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	entry := buildstore.BuildDataDirName + "/"

	path := filepath.Join(u.HomeDir, name)
	data, err := ioutil.ReadFile(path)
	if os.IsNotExist(err) {
		err = nil
	} else if bytes.Contains(data, []byte("\n"+entry+"\n")) {
		// already has entry
		return
	}

	data = append(data, []byte("\n\n# Sourcegraph build data\n"+entry+"\n")...)
	err = ioutil.WriteFile(path, data, 0700)
	if err != nil {
		log.Fatal(err)
	}
}
