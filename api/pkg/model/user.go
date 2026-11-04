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
	Roles     []Role
	Groups    []string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Validate checks the user invariants that do not need a store lookup.
func (u User) Validate() error {
	if err := ValidateUserName(u.Name); err != nil {
		return err
	}

	for _, role := range u.Roles {
		if !role.Valid() {
			return &ValidationError{Field: "roles", Reason: "unknown role " + string(role)}
		}
	}

	return nil
}

// HasRole reports whether the user holds role.
func (u User) HasRole(role Role) bool {
	for _, held := range u.Roles {
		if held == role {
			return true
		}
	}

	return false
}
