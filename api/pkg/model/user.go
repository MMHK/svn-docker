package model

import (
	"regexp"
	"time"
)

// userNamePattern is deliberately conservative: it must round-trip through
// htpasswd and the authz file without escaping.
var userNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

// User is an SVN account backed by the htpasswd file.
type User struct {
	Name      string
	Comment   string
	Groups    []string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Validate checks the user invariants that do not need a store lookup.
func (u User) Validate() error {
	return ValidateUserName(u.Name)
}
