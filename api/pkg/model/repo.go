package model

import (
	"regexp"
	"strings"
	"time"
)

// repoNamePattern is the repository name contract from the rewrite plan
// (section 5.4). A leading dot is rejected separately.
var repoNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// Repository is a Subversion repository living under one of the configured
// SVNParentPath directories.
type Repository struct {
	Name              string
	Path              string
	Parent            string
	URL               string
	IsRepository      bool
	HeadRevision      int64
	LastChangedRev    int64
	LastChangedAuthor string
	LastChangedAt     time.Time
}

// Validate checks the repository name contract.
//
// The absolute Path is intentionally not validated here; filesystem
// containment (Clean + prefix + Lstat) belongs to the store layer.
func (r Repository) Validate() error {
	return ValidateRepositoryName(r.Name)
}

// ValidateRepositoryName applies the plan section 5.4 name contract: 1-64
// letters, digits, '_' or '-', and no leading dot (which would make the
// directory hidden and could collide with `.svn`).
func ValidateRepositoryName(name string) error {
	name = strings.TrimSpace(name)

	if name == "" {
		return &ValidationError{Field: "name", Reason: "must not be empty"}
	}

	if strings.HasPrefix(name, ".") {
		return &ValidationError{Field: "name", Reason: "must not start with a dot"}
	}

	if !repoNamePattern.MatchString(name) {
		return &ValidationError{
			Field:  "name",
			Reason: "must contain only letters, digits, '_' or '-' (max 64)",
		}
	}

	return nil
}
