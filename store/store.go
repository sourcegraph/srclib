package store

import (
	"errors"
	"os"
)

// IsNotExist returns a boolean indicating whether err is known to
// report that an item (def/unit/etc.) does not exist.
func IsNotExist(err error) bool {
	return err == errDefNotExist || err == errUnitNotExist || err == errVersionNotExist || err == errRepoNotExist
}

var (
	errDefNotExist     = errors.New("def does not exist")
	errUnitNotExist    = errors.New("unit does not exist")
	errVersionNotExist = errors.New("version does not exist")
	errRepoNotExist    = errors.New("repo does not exist")
)

// An InvalidKeyError occurs when an invalid key is passed to a store
// or filter. E.g., passing a DefKey with an empty Path to
// (UnitStore).Def or ByDefKeyFilter triggers an InvalidKeyError.
type InvalidKeyError struct{ msg string }

func (e *InvalidKeyError) Error() string { return e.msg }

func isInvalidKey(err error) bool {
	switch err.(type) {
	case *InvalidKeyError:
		return true
	}
	return false
}

// isStoreNotExist returns a boolean indicating whether err is known
// to report that a store does not exist. It can be used to determine
// whether a "not exists" error should be returned in combined stores
// (repoStores, and treeStores, and unitStores types) that open
// lower-level stores during lookups.
func isStoreNotExist(err error) bool {
	if err == nil {
		return false
	}
	return os.IsNotExist(err) || err == errRepoNoInit || err == errTreeNoInit || err == errMultiRepoStoreNoInit || err == errUnitNoInit
}
