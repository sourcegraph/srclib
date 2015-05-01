package src

import (
	"fmt"
	"log"
	"os"
	"time"

	"golang.org/x/net/context"

	"sourcegraph.com/sourcegraph/go-flags"
	"sourcegraph.com/sourcegraph/go-sourcegraph/sourcegraph"
	"sourcegraph.com/sqs/pbtypes"
)

func initRemoteImportBuildCmd(remoteGroup *flags.Command) {
	importBuildCmd, err := remoteGroup.AddCommand("import-build",
		"tell a remote to import a build for a repository at a specific commit",
		"The import-build command tells the remote to import build data for a repository at a specific commit. To import build data that was produced locally, first run `src build-data upload` (or run `src push`, which performs both steps).",
		&remoteImportBuildCmd,
	)
	if err != nil {
		log.Fatal(err)
	}
	if lrepo, err := openLocalRepo(); err == nil {
		SetOptionDefaultValue(importBuildCmd.Group, "commit", lrepo.CommitID)
	}
}

type RemoteImportBuildCmd struct {
	CommitID string `short:"c" long:"commit" description:"commit ID of data to import" required:"yes"`
}

var remoteImportBuildCmd RemoteImportBuildCmd

func (c *RemoteImportBuildCmd) Execute(args []string) error {
	cl := Client()
	defer cl.Close()

	if GlobalOpt.Verbose {
		log.Printf("Creating a new import-only build for repo %q commit %q", remoteCmd.RepoURI, c.CommitID)
	}

	repo, err := cl.Repos.Get(context.TODO(), &sourcegraph.RepoSpec{URI: remoteCmd.RepoURI})
	if err != nil {
		return err
	}

	repoSpec := sourcegraph.RepoSpec{URI: remoteCmd.RepoURI}
	repoRevSpec := sourcegraph.RepoRevSpec{RepoSpec: repoSpec, Rev: c.CommitID}

	// Resolve to the full commit ID, and ensure that the remote
	// server knows about the commit.
	commit, err := cl.Repos.GetCommit(context.TODO(), &repoRevSpec)
	if err != nil {
		return err
	}
	repoRevSpec.CommitID = string(commit.ID)

	build, err := cl.Builds.Create(context.TODO(), &sourcegraph.BuildsCreateOp{RepoRev: repoRevSpec, Opt: &sourcegraph.BuildCreateOptions{
		BuildConfig: sourcegraph.BuildConfig{
			Import: true,
			Queue:  false,
		},
		Force: true,
	}})

	if err != nil {
		return err
	}
	if GlobalOpt.Verbose {
		log.Printf("Created build #%d", build.BID)
	}

	now := pbtypes.NewTimestamp(time.Now())
	host := fmt.Sprintf("local (USER=%s)", os.Getenv("USER"))
	buildUpdate := sourcegraph.BuildUpdate{StartedAt: &now, Host: host}
	if _, err := cl.Builds.Update(context.TODO(), &sourcegraph.BuildsUpdateOp{Build: build.Spec(), Info: buildUpdate}); err != nil {
		return err
	}

	importTask := &sourcegraph.BuildTask{
		BID:   build.BID,
		Repo:  repoSpec.URI,
		Op:    sourcegraph.ImportTaskOp,
		Queue: true,
	}
	tasks, err := cl.Builds.CreateTasks(context.TODO(), &sourcegraph.BuildsCreateTasksOp{Build: build.Spec(), Tasks: []*sourcegraph.BuildTask{importTask}})
	if err != nil {
		return err
	}
	importTask = tasks.BuildTasks[0]
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
				logs, err := cl.Builds.GetTaskLog(context.TODO(), &sourcegraph.BuildsGetTaskLogOp{Task: importTask.Spec(), Opt: &logOpt})
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

		tasks, err := cl.Builds.ListBuildTasks(context.TODO(), &sourcegraph.BuildsListBuildTasksOp{Build: build.Spec()})
		if err != nil {
			return err
		}
		importTask = nil
		for _, task := range tasks.BuildTasks {
			if task.TaskID == taskID {
				importTask = task
				break
			}
		}
		if importTask == nil {
			return fmt.Errorf("task #%d not found in task list for build #%d", taskID, build.BID)
		}

		if !started && importTask.StartedAt != nil {
			log.Printf("# Import started.")
			started = true
		}

		if importTask.EndedAt != nil {
			if importTask.Success {
				log.Printf("# Import succeeded!")
			} else if importTask.Failure {
				log.Printf("# Import failed!")
				return fmt.Errorf("import failed")
			}
			break
		}

		time.Sleep(time.Duration(i) * 200 * time.Millisecond)
	}

	log.Printf("# View the repository at:")
	log.Printf("# %s://%s/%s@%s", cl.BaseURL.Scheme, cl.BaseURL.Host, repo.URI, repoRevSpec.Rev)

	return nil
}
