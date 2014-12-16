package src

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"sourcegraph.com/sourcegraph/go-sourcegraph/sourcegraph"
	"sourcegraph.com/sourcegraph/srclib/buildstore"
)

func init() {
	pushGroup, err := CLI.AddCommand("push",
		"build, upload, and import the current commit (to make it available on Sourcegraph.com)",
		"The push command (1) builds the current commit if it's not built; (2) uploads the current repository commit's build data; and (3) imports it into Sourcegraph.",
		&pushCmd,
	)
	if err != nil {
		log.Fatal(err)
	}
	if repo := openCurrentRepo(); repo != nil {
		SetOptionDefaultValue(pushGroup.Group, "commit", repo.CommitID)
	}

	remoteGroup, err := CLI.AddCommand("remote",
		"remote operations",
		"The remote command contains subcommands perform operations on Sourcegraph.com.",
		&remoteCmd,
	)
	if err != nil {
		log.Fatal(err)
	}
	if repo := openCurrentRepo(); repo != nil {
		SetOptionDefaultValue(remoteGroup.Group, "repo", repo.URI())
	}

	importBuildDataCmd, err := remoteGroup.AddCommand("import-build-data",
		"import build data for a repository at a specific commit",
		"The `src remote import-build-data` subcommand imports build data for a repository at a specific commit. To import build data that was produced locally, first run `src build-data upload` (or run `src push`, which performs both steps).",
		&remoteImportBuildDataCmd,
	)
	if err != nil {
		log.Fatal(err)
	}
	if repo := openCurrentRepo(); repo != nil {
		SetOptionDefaultValue(importBuildDataCmd.Group, "commit", repo.CommitID)
	}

	initRemoteBuildCmds(remoteGroup)
	initRemoteRepoCmds(remoteGroup)
}

type PushCmd struct {
	Dir      Directory `short:"C" long:"directory" description:"change to DIR before doing anything" value-name:"DIR"`
	CommitID string    `short:"c" long:"commit" description:"commit ID of data to import" required:"yes"`
}

var pushCmd PushCmd

func (c *PushCmd) Execute(args []string) error {
	if c.Dir != "" {
		if err := os.Chdir(string(c.Dir)); err != nil {
			return err
		}
	}

	repo, err := OpenRepo(".")
	if err != nil {
		return err
	}
	buildStore, err := buildstore.LocalRepo(repo.RootDir)
	if err != nil {
		return err
	}
	if err := ensureBuild(buildStore, repo); err != nil {
		return fmt.Errorf("local build failed: %s", err)
	}

	cl := NewAPIClientWithAuthIfPresent()

	repoSpec := sourcegraph.RepoSpec{URI: remoteCmd.RepoURI}
	repoRevSpec := sourcegraph.RepoRevSpec{RepoSpec: repoSpec, Rev: c.CommitID}

	if _, _, err := cl.Repos.GetOrCreate(repo.RepoRevSpec().RepoSpec, nil); err != nil {
		return fmt.Errorf("couldn't find repo %q on remote: %s", repo.URI(), err)
	}
	if _, err := getCommitWithRefreshAndRetry(cl, repoRevSpec); err != nil {
		return err
	}

	if err := buildDataUploadCmd.Execute(nil); err != nil {
		return err
	}
	if err := remoteImportBuildDataCmd.Execute(nil); err != nil {
		return err
	}
	return nil
}

// getCommitWithRefreshAndRetry tries to get a repository commit. If
// it doesn't exist, it triggers a refresh of the repo's VCS data and
// then retries (until maxGetCommitVCSRefreshWait has elapsed).
func getCommitWithRefreshAndRetry(cl *sourcegraph.Client, repoRevSpec sourcegraph.RepoRevSpec) (*sourcegraph.Commit, error) {
	timeout := time.After(maxGetCommitVCSRefreshWait)
	done := make(chan struct{})
	var commit *sourcegraph.Commit
	var err error
	go func() {
		refreshTriggered := false
		for {
			commit, _, err = cl.Repos.GetCommit(repoRevSpec, nil)

			// Keep retrying if it's a 404, but stop trying if we succeeded, or if it's some other
			// error.
			if !sourcegraph.IsHTTPErrorCode(err, http.StatusNotFound) {
				break
			}

			if !refreshTriggered {
				_, err = cl.Repos.RefreshVCSData(repoRevSpec.RepoSpec)
				if err != nil {
					err = fmt.Errorf("failed to trigger VCS refresh for repo %s: %s", repoRevSpec.URI, err)
					break
				}
				log.Printf("Repository %s revision %s wasn't found on remote. Triggered refresh of VCS data; waiting %s.", repoRevSpec.URI, repoRevSpec.Rev, maxGetCommitVCSRefreshWait)
				refreshTriggered = true
			}
			time.Sleep(time.Second)
		}
		done <- struct{}{}
	}()
	select {
	case <-done:
		return commit, err
	case <-timeout:
		return nil, fmt.Errorf("repo %s revision %s not found on remote, even after triggering a VCS refresh and waiting %s (if you are sure that commit has been pushed, try again later)", repoRevSpec.URI, repoRevSpec.Rev, maxGetCommitVCSRefreshWait)
	}
}

const maxGetCommitVCSRefreshWait = time.Second * 10

type RemoteCmd struct {
	RepoURI string `short:"r" long:"repo" description:"repository URI (defaults to VCS 'srclib' or 'origin' remote URL)" required:"yes"`
}

var remoteCmd RemoteCmd

func (c *RemoteCmd) Execute(args []string) error {
	return nil
}

type RemoteImportBuildDataCmd struct {
	CommitID string `short:"c" long:"commit" description:"commit ID of data to import" required:"yes"`
}

var remoteImportBuildDataCmd RemoteImportBuildDataCmd

func (c *RemoteImportBuildDataCmd) Execute(args []string) error {
	cl := NewAPIClientWithAuthIfPresent()

	if GlobalOpt.Verbose {
		log.Printf("Creating a new import-only build for repo %q commit %q", remoteCmd.RepoURI, c.CommitID)
	}

	repo, _, err := cl.Repos.GetOrCreate(sourcegraph.RepoSpec{URI: remoteCmd.RepoURI}, nil)
	if err != nil {
		return err
	}

	repoSpec := sourcegraph.RepoSpec{URI: remoteCmd.RepoURI}
	repoRevSpec := sourcegraph.RepoRevSpec{RepoSpec: repoSpec, Rev: c.CommitID}

	// Resolve to the full commit ID, and ensure that the remote
	// server knows about the commit.
	commit, err := getCommitWithRefreshAndRetry(cl, repoRevSpec)
	if err != nil {
		return err
	}
	repoRevSpec.CommitID = string(commit.ID)

	build, _, err := cl.Builds.Create(repoSpec, &sourcegraph.BuildCreateOptions{
		BuildConfig: sourcegraph.BuildConfig{
			Import:   true,
			Queue:    false,
			CommitID: repoRevSpec.CommitID,
		},
		Force: true,
	})
	if err != nil {
		return err
	}
	if GlobalOpt.Verbose {
		log.Printf("Created build #%d", build.BID)
	}

	now := time.Now()
	host := fmt.Sprintf("local (USER=%s)", os.Getenv("USER"))
	buildUpdate := sourcegraph.BuildUpdate{StartedAt: &now, Host: &host}
	if _, _, err := cl.Builds.Update(build.Spec(), buildUpdate); err != nil {
		return err
	}

	importTask := &sourcegraph.BuildTask{
		BID:   build.BID,
		Op:    sourcegraph.ImportTaskOp,
		Queue: true,
	}
	tasks, _, err := cl.Builds.CreateTasks(build.Spec(), []*sourcegraph.BuildTask{importTask})
	if err != nil {
		return err
	}
	importTask = tasks[0]
	if GlobalOpt.Verbose {
		log.Printf("Created import task #%d", importTask.TaskID)
	}

	// Stream logs.
	done := make(chan struct{})
	go func() {
		var logOpt sourcegraph.BuildGetLogOptions
		loopsSinceLastLog := 0
		for {
			select {
			case <-done:
				return
			case <-time.After(time.Duration(loopsSinceLastLog+1) * 500 * time.Millisecond):
				logs, _, err := cl.Builds.GetTaskLog(importTask.Spec(), &logOpt)
				if err != nil {
					log.Printf("Warning: failed to get build logs: %s.", err)
					return
				}
				if len(logs.Entries) == 0 {
					loopsSinceLastLog++
					continue
				}
				logOpt.MinID = logs.MaxID
				for _, e := range logs.Entries {
					fmt.Println(e)
				}
				loopsSinceLastLog = 0
			}
		}
	}()

	defer func() {
		done <- struct{}{}
	}()
	taskID := importTask.TaskID
	started := false
	log.Printf("# Import queued. Waiting for task #%d in build #%d to start...", importTask.TaskID, build.BID)
	for i, start := 0, time.Now(); ; i++ {
		if time.Since(start) > 45*time.Minute {
			return fmt.Errorf("import timed out after %s", time.Since(start))
		}

		tasks, _, err := cl.Builds.ListBuildTasks(build.Spec(), nil)
		if err != nil {
			return err
		}
		importTask = nil
		for _, task := range tasks {
			if task.TaskID == taskID {
				importTask = task
				break
			}
		}
		if importTask == nil {
			return fmt.Errorf("task #%d not found in task list for build #%d", taskID, build.BID)
		}

		if !started && importTask.StartedAt.Valid {
			log.Printf("# Import started.")
			started = true
		}

		if importTask.EndedAt.Valid {
			if importTask.Success {
				log.Printf("# Import succeeded!")
			} else if importTask.Failure {
				log.Printf("# Import failed!")
			}
			break
		}

		time.Sleep(time.Duration(i) * 200 * time.Millisecond)
	}

	log.Printf("# View the repository at:")
	log.Printf("# %s://%s/%s@%s", cl.BaseURL.Scheme, cl.BaseURL.Host, repo.URI, repoRevSpec.Rev)

	return nil
}
