package src

import (
	"fmt"
	"log"
	"os"
	"time"

	"sourcegraph.com/sourcegraph/go-sourcegraph/sourcegraph"
	"sourcegraph.com/sourcegraph/srclib/buildstore"
)

func init() {
	_, err := CLI.AddCommand("push",
		"build, upload, and import the current commit (to make it available on Sourcegraph.com)",
		"The push command (1) builds the current commit if it's not built; (2) uploads the current repository commit's build data; and (3) imports it into Sourcegraph.",
		&pushCmd,
	)
	if err != nil {
		log.Fatal(err)
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
	Dir Directory `short:"C" long:"directory" description:"change to DIR before doing anything" value-name:"DIR"`
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

	apiclient := NewAPIClientWithAuthIfPresent()

	if _, _, err := apiclient.Repos.GetOrCreate(repo.RepoRevSpec().RepoSpec, nil); err != nil {
		return fmt.Errorf("couldn't find repo %q on remote: %s", repo.URI(), err)
	}
	if _, _, err := apiclient.Repos.GetCommit(repo.RepoRevSpec(), nil); err != nil {
		if _, err := apiclient.Repos.RefreshVCSData(repo.RepoRevSpec().RepoSpec); err != nil {
			log.Printf("Warning: failed to trigger VCS update: %s.", err)
		}
		return fmt.Errorf("could not find commit %s on remote (%s); was it pushed? a VCS update was just triggered, so try again in a few seconds/minutes", repo.RepoRevSpec().CommitID, err)
	}

	if err := buildDataUploadCmd.Execute(nil); err != nil {
		return err
	}
	if err := remoteImportBuildDataCmd.Execute(nil); err != nil {
		return err
	}
	return nil
}

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
	repo, err := OpenRepo(".")
	if err != nil {
		return err
	}

	if GlobalOpt.Verbose {
		log.Printf("Creating a new import-only build for repo %q commit %q", repo.URI(), repo.CommitID)
	}

	repoSpec := sourcegraph.RepoSpec{URI: repo.URI()}
	repoRevSpec := sourcegraph.RepoRevSpec{RepoSpec: repoSpec, Rev: repo.CommitID, CommitID: repo.CommitID}

	apiclient := NewAPIClientWithAuthIfPresent()

	rrepo, _, err := apiclient.Repos.Get(repoSpec, nil)
	if err != nil {
		return err
	}

	// Check that the remote server knows about the commit.
	if _, _, err := apiclient.Repos.GetCommit(repoRevSpec, nil); err != nil {
		return fmt.Errorf("could not find commit %s on remote (%s); was it pushed?", repoRevSpec.CommitID, err)
	}

	build, _, err := apiclient.Builds.Create(repoSpec, &sourcegraph.BuildCreateOptions{
		BuildConfig: sourcegraph.BuildConfig{
			Import:   true,
			Queue:    false,
			CommitID: repo.CommitID,
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
	if _, _, err := apiclient.Builds.Update(build.Spec(), buildUpdate); err != nil {
		return err
	}

	importTask := &sourcegraph.BuildTask{
		BID:   build.BID,
		Op:    sourcegraph.ImportTaskOp,
		Queue: true,
	}
	tasks, _, err := apiclient.Builds.CreateTasks(build.Spec(), []*sourcegraph.BuildTask{importTask})
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
				logs, _, err := apiclient.Builds.GetTaskLog(importTask.Spec(), &logOpt)
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

		tasks, _, err := apiclient.Builds.ListBuildTasks(build.Spec(), nil)
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
	log.Printf("# %s://%s/%s@%s", apiclient.BaseURL.Scheme, apiclient.BaseURL.Host, rrepo.URI, build.CommitID)

	return nil
}
