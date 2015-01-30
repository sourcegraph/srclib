package store

import "os"

// isStoreNotExist returns a boolean indicating whether err is known
// to report that a store does not exist. It can be used to determine
// whether a "not exists" error should be returned in combined stores
// (repoStores, and treeStores, and unitStores types) that open
// lower-level stores during lookups.
func isStoreNotExist(err error) bool {
	if err == nil {
		return false
	}
	if _, ok := err.(*errIndexNotExist); ok {
		return true
	}
	return os.IsNotExist(err) || err == errRepoNoInit || err == errTreeNoInit || err == errMultiRepoStoreNoInit || err == errUnitNoInit
}
