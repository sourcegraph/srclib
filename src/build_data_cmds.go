package src

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/kr/fs"
	"golang.org/x/tools/godoc/vfs"

	"code.google.com/p/rog-go/parallel"

	"sourcegraph.com/sourcegraph/go-sourcegraph/router"
	"sourcegraph.com/sourcegraph/go-sourcegraph/sourcegraph"
	"sourcegraph.com/sourcegraph/rwvfs"
	"sourcegraph.com/sourcegraph/srclib/buildstore"
)

func init() {
	buildDataGroup, err := CLI.AddCommand("build-data",
		"build data operations",
		"The build-data command contains subcommands for performing operations on local and remote build data.",
		&buildDataCmd,
	)
	if err != nil {
		log.Fatal(err)
	}

	_, err = buildDataGroup.AddCommand("ls",
		"list build data files and dirs",
		"The `src build-data ls` subcommand lists build data files and directories for a repository at a specific commit.",
		&buildDataListCmd,
	)
	if err != nil {
		log.Fatal(err)
	}

	_, err = buildDataGroup.AddCommand("cat",
		"display contents of build files",
		"The `src build-data cat` subcommand displays the contents of a build data file for a repository at a specific commit.",
		&buildDataCatCmd,
	)
	if err != nil {
		log.Fatal(err)
	}

	_, err = buildDataGroup.AddCommand("rm",
		"remove build data files and dirs",
		"The `src build-data rm` subcommand removes a build data file or directory for a repository at a specific commit.",
		&buildDataRemoveCmd,
	)
	if err != nil {
		log.Fatal(err)
	}

	_, err = buildDataGroup.AddCommand("fetch",
		"fetch remote build data to local dir",
		"Fetch remote build data (from Sourcegraph.com) for the current repository to the local .srclib-cache directory.",
		&buildDataFetchCmd,
	)
	if err != nil {
		log.Fatal(err)
	}

	_, err = buildDataGroup.AddCommand("upload",
		"upload local build data to remote",
		"Upload local build data (in .srclib-cache) for the current repository to Sourcegraph.com.",
		&buildDataUploadCmd,
	)
	if err != nil {
		log.Fatal(err)
	}
}

type BuildDataCmd struct {
	Local bool `short:"l" long:"local" description:"execute build-data ls/cat/rm subcommands on local build data (.srclib-cache), not remote (sourcegraph.com)"`
}

var buildDataCmd BuildDataCmd

func (c *BuildDataCmd) Execute(args []string) error { return nil }

func getBuildDataCmdFS(repo *Repo) (rwvfs.FileSystem, error) {
	if buildDataCmd.Local {
		localStore, err := buildstore.LocalRepo(repo.RootDir)
		if err != nil {
			return nil, err
		}
		return localStore.Commit(repo.CommitID), nil
	}
	return NewAPIClientWithAuthIfPresent().BuildData.FileSystem(repo.RepoRevSpec())
}

type BuildDataListCmd struct {
	Args struct {
		Dir string `name:"DIR" default:"." description:"list build data files in this dir"`
	} `positional-args:"yes"`

	Recursive bool   `short:"r" long:"recursive" description:"list recursively"`
	Long      bool   `short:"l" long:"long" description:"show file sizes and times"`
	Type      string `long:"type" description:"show only entries of this type (f=file, d=dir)"`
	URLs      bool   `long:"urls" description:"show URLs to build data files (implies -l)"`
}

var buildDataListCmd BuildDataListCmd

func (c *BuildDataListCmd) Execute(args []string) error {
	if c.URLs && buildDataCmd.Local {
		return fmt.Errorf("using --urls is incompatible with the build-data -l/--local option because local build data files do not have a URL")
	}
	if c.URLs {
		c.Long = true
	}

	repo, err := OpenRepo(".")
	if err != nil {
		return err
	}

	dir := c.Args.Dir
	if dir == "" {
		dir = "."
	}

	if GlobalOpt.Verbose {
		log.Printf("Listing build files for repository %q commit %q in dir %q", repo.URI(), repo.CommitID, dir)
	}

	bdfs, err := getBuildDataCmdFS(repo)
	if err != nil {
		return err
	}

	printFile := func(fi os.FileInfo) {
		if c.Type == "f" && !fi.Mode().IsRegular() {
			return
		}
		if c.Type == "d" && !fi.Mode().IsDir() {
			return
		}

		var suffix string
		if fi.IsDir() {
			suffix = "/"
		}

		var url string
		if c.URLs {
			spec := sourcegraph.BuildDataFileSpec{RepoRev: repo.RepoRevSpec(), Path: filepath.Join(dir, fi.Name())}
			u := router.URITo(router.RepoBuildDataEntry, router.MapToArray(spec.RouteVars())...)
			endpointURL := getEndpointURL()
			u.Host = endpointURL.Host
			u.Scheme = endpointURL.Scheme
			url = u.String()
		}

		if c.Long {
			var timeStr string
			if !fi.ModTime().IsZero() {
				timeStr = fi.ModTime().Format("Jan _2 15:04")
			}
			fmt.Printf("% 7d %12s %s%s %s\n", fi.Size(), timeStr, fi.Name(), suffix, url)
		} else {
			fmt.Println(fi.Name() + suffix)
		}
	}

	var fis []os.FileInfo
	if c.Recursive {
		w := fs.WalkFS(dir, rwvfs.Walkable(bdfs))
		for w.Step() {
			if err := w.Err(); err != nil {
				return err
			}
			printFile(treeFileInfo{w.Path(), w.Stat()})
		}
	} else {
		fis, err = bdfs.ReadDir(dir)
		if err != nil {
			return err
		}
		for _, fi := range fis {
			printFile(fi)
		}
	}

	return nil
}

type treeFileInfo struct {
	path string
	os.FileInfo
}

func (fi treeFileInfo) Name() string { return fi.path }

type BuildDataCatCmd struct {
	Args struct {
		File string `name:"FILE" default:"." description:"file whose contents to print"`
	} `positional-args:"yes"`
}

var buildDataCatCmd BuildDataCatCmd

func (c *BuildDataCatCmd) Execute(args []string) error {
	repo, err := OpenRepo(".")
	if err != nil {
		return err
	}

	file := c.Args.File
	if file == "" {
		return fmt.Errorf("no file specified")
	}

	if GlobalOpt.Verbose {
		log.Printf("Displaying build file %q for repository %q commit %q", file, repo.URI(), repo.CommitID)
	}

	bdfs, err := getBuildDataCmdFS(repo)
	if err != nil {
		return err
	}

	f, err := bdfs.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(os.Stdout, f)
	return err
}

type BuildDataRemoveCmd struct {
	Recursive bool `short:"r" description:"recursively delete files and dir"`
	Args      struct {
		Files []string `name:"FILES" default:"." description:"file to remove"`
	} `positional-args:"yes"`
}

var buildDataRemoveCmd BuildDataRemoveCmd

func (c *BuildDataRemoveCmd) Execute(args []string) error {
	repo, err := OpenRepo(".")
	if err != nil {
		return err
	}

	if len(c.Args.Files) == 0 {
		return fmt.Errorf("no files specified")
	}

	if GlobalOpt.Verbose {
		log.Printf("Removing build files %v for repository %q commit %q", c.Args.Files, repo.URI(), repo.CommitID)
	}

	bdfs, err := getBuildDataCmdFS(repo)
	if err != nil {
		return err
	}

	vfs := removeLoggedFS{rwvfs.Walkable(bdfs)}

	for _, file := range c.Args.Files {
		if c.Recursive {
			if err := buildstore.RemoveAll(file, vfs); err != nil {
				return err
			}
		} else {
			if err := vfs.Remove(file); err != nil {
				return err
			}
		}
	}
	return nil
}

type removeLoggedFS struct{ rwvfs.WalkableFileSystem }

func (fs removeLoggedFS) Remove(path string) error {
	if err := fs.WalkableFileSystem.Remove(path); err != nil {
		return err
	}
	if GlobalOpt.Verbose {
		log.Printf("Removed %s", path)
	}
	return nil
}

type BuildDataFetchCmd struct {
	DryRun bool `short:"n" long:"dry-run" description:"don't do anything, just show what would be done"`
}

var buildDataFetchCmd BuildDataFetchCmd

func (c *BuildDataFetchCmd) Execute(args []string) error {
	repo, err := OpenRepo(".")
	if err != nil {
		return err
	}
	localStore, err := buildstore.LocalRepo(repo.RootDir)
	if err != nil {
		log.Fatal(err)
	}
	localFS := localStore.Commit(repo.CommitID)
	if err := rwvfs.MkdirAll(localFS, "."); err != nil {
		return err
	}

	apiclient := NewAPIClientWithAuthIfPresent()

	if _, _, err := apiclient.Repos.Get(repo.RepoRevSpec().RepoSpec, nil); err != nil {
		return fmt.Errorf("couldn't find repository on remote: %s.", err)
	}

	// Use uncached API client because the .srclib-cache already
	// caches it, and we want to be able to stream large files.
	//
	// TODO(sqs): this uncached client isn't authed because it doesn't
	// have the other API client's http.Client or http.RoundTripper
	apiclientUncached := sourcegraph.NewClient(nil)
	apiclientUncached.BaseURL = apiclient.BaseURL
	remoteFS, err := apiclientUncached.BuildData.FileSystem(repo.RepoRevSpec())
	if err != nil {
		return err
	}

	if GlobalOpt.Verbose {
		log.Printf("Fetching remote build files for repository %q commit %q from %s to %s...", repo.URI(), repo.CommitID, remoteFS.String(), localFS.String())
	}

	// TODO(sqs): check if file exists in local cache and don't fetch it if it does and if it is identical

	par := parallel.NewRun(8)
	w := fs.WalkFS(".", rwvfs.Walkable(remoteFS))
	for w.Step() {
		if err := w.Err(); err != nil {
			return err
		}
		fi := w.Stat()
		if fi == nil {
			continue
		}
		if !fi.Mode().IsRegular() {
			continue
		}
		path := w.Path()
		par.Do(func() error {
			return fetchFile(remoteFS, localFS, path, fi, c.DryRun)
		})
	}
	return par.Wait()
}

func fetchFile(remote vfs.FileSystem, local rwvfs.FileSystem, path string, fi os.FileInfo, dryRun bool) error {
	kb := float64(fi.Size()) / 1024
	if GlobalOpt.Verbose || dryRun {
		log.Printf("Fetching %s (%.1fkb)", path, kb)
	}
	if dryRun {
		return nil
	}

	if err := rwvfs.MkdirAll(local, filepath.Dir(path)); err != nil {
		return err
	}

	rf, err := remote.Open(path)
	if err != nil {
		return err
	}
	defer rf.Close()

	lf, err := local.Create(path)
	if err != nil {
		return err
	}
	defer lf.Close()

	if _, err := io.Copy(lf, rf); err != nil {
		return err
	}

	if GlobalOpt.Verbose {
		log.Printf("Fetched %s (%.1fkb)", path, kb)
	}

	return lf.Close()
}

type BuildDataUploadCmd struct {
	DryRun bool `short:"n" long:"dry-run" description:"don't do anything, just show what would be done"`
}

var buildDataUploadCmd BuildDataUploadCmd

func (c *BuildDataUploadCmd) Execute(args []string) error {
	repo, err := OpenRepo(".")
	if err != nil {
		return err
	}
	localStore, err := buildstore.LocalRepo(repo.RootDir)
	if err != nil {
		return err
	}
	localFS := localStore.Commit(repo.CommitID)

	apiclient := NewAPIClientWithAuthIfPresent()

	if _, _, err := apiclient.Repos.Get(repo.RepoRevSpec().RepoSpec, nil); err != nil {
		return fmt.Errorf("couldn't find repository on remote: %s.", err)
	}

	remoteFS, err := apiclient.BuildData.FileSystem(repo.RepoRevSpec())
	if err != nil {
		return err
	}

	if GlobalOpt.Verbose {
		log.Printf("Uploading local build files to remote for repository %q commit %q from %s to %s...", repo.URI(), repo.CommitID, localFS.String(), remoteFS.String())
	}

	// TODO(sqs): check if file exists remotely and don't upload it if it does and if it is identical

	par := parallel.NewRun(8)
	w := fs.WalkFS(".", rwvfs.Walkable(localFS))
	for w.Step() {
		if err := w.Err(); err != nil {
			return err
		}
		fi := w.Stat()
		if fi == nil {
			continue
		}
		if !fi.Mode().IsRegular() {
			continue
		}
		path := w.Path()
		par.Do(func() error {
			return uploadFile(localFS, remoteFS, path, fi, c.DryRun)
		})
	}
	return par.Wait()
}

func uploadFile(local vfs.FileSystem, remote rwvfs.FileSystem, path string, fi os.FileInfo, dryRun bool) error {
	kb := float64(fi.Size()) / 1024
	if GlobalOpt.Verbose || dryRun {
		log.Printf("Uploading %s (%.1fkb)", path, kb)
	}
	if dryRun {
		return nil
	}

	lf, err := local.Open(path)
	if err != nil {
		return err
	}

	if err := rwvfs.MkdirAll(remote, filepath.Dir(path)); err != nil {
		return err
	}
	rf, err := remote.Create(path)
	if err != nil {
		return err
	}
	defer rf.Close()

	if _, err := io.Copy(rf, lf); err != nil {
		return err
	}

	if GlobalOpt.Verbose {
		log.Printf("Uploaded %s (%.1fkb)", path, kb)
	}

	return rf.Close()
}
