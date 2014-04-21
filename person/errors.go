package person

import (
	"errors"
	"fmt"
)

// ErrNotExist is an error indicating that no such user exists.
var ErrNotExist = errors.New("user does not exist")

// ErrRenamed is an error type that indicates that a user account was renamed
// from OldLogin to NewLogin.
type ErrRenamed struct {
	// OldLogin is the previous login name.
	OldLogin string

	// NewLogin is what the old login was renamed to.
	NewLogin string
}

func (e ErrRenamed) Error() string {
	return fmt.Sprintf("login %q was renamed to %q; use the new name", e.OldLogin, e.NewLogin)
}
