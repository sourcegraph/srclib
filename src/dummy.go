package src

import "sourcegraph.com/sourcegraph/rwvfs"

// TODO(sqs!nodb-ctx): remove these impls when it compiles again

func getBuildDataFS(local bool, repo, commitID string) (rwvfs.FileSystem, string, error) {
	panic("not implemented")
}
