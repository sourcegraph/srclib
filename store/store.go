package store

import "errors"

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
