package model

import (
	"regexp"
	"strings"
)

// groupNamePattern is the group name rule from plan section 5.4: letters,
// digits, '_' and '-', at most 64 characters.
var groupNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// Group is an SVN authz group.
type Group struct {
	Name        string
	Description string
	Members     []string
}

// Validate checks the group invariants that do not need a store lookup.
func (g Group) Validate() error {
	name := strings.TrimSpace(g.Name)
	if name == "" {
		return &ValidationError{Field: "name", Reason: "must not be empty"}
	}

	if !groupNamePattern.MatchString(name) {
		return &ValidationError{
			Field:  "name",
			Reason: "must contain only letters, digits, '_' or '-' (max 64)",
		}
	}

	for _, member := range g.Members {
		if !userNamePattern.MatchString(member) {
			return &ValidationError{Field: "members", Reason: "invalid user name " + member}
		}
	}

	return nil
}
