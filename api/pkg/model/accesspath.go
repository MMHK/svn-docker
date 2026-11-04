package model

import (
	"regexp"
	"strings"
	"unicode"
)

// accessPathPattern is `repo:/path` with a repository-name prefix.
var accessPathPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}:/.*$`)

// ACLEntry grants a permission to a user, a `@group` or `*`.
type ACLEntry struct {
	Principal  string
	Permission Permission
}

// AccessPath is one authz section such as `project:/trunk`.
type AccessPath struct {
	Path string
	ACL  []ACLEntry
}

// Validate checks the access-path contract from the rewrite plan (section 5.4).
func (a AccessPath) Validate() error {
	path := a.Path
	if path == "" {
		return &ValidationError{Field: "path", Reason: "must not be empty"}
	}

	if !accessPathPattern.MatchString(path) {
		return &ValidationError{Field: "path", Reason: "must look like <repo>:/<path>"}
	}

	if strings.Contains(path, "..") {
		return &ValidationError{Field: "path", Reason: "must not contain '..'"}
	}

	if strings.ContainsAny(path, "\\\x00") {
		return &ValidationError{Field: "path", Reason: "must not contain backslashes or NUL"}
	}

	for _, r := range path {
		if unicode.IsControl(r) {
			return &ValidationError{Field: "path", Reason: "must not contain control characters"}
		}
	}

	for _, entry := range a.ACL {
		if !entry.Permission.Valid() {
			return &ValidationError{Field: "acl", Reason: "unknown permission " + string(entry.Permission)}
		}

		if !ValidPrincipal(entry.Principal) {
			return &ValidationError{Field: "acl", Reason: "invalid principal " + entry.Principal}
		}
	}

	return nil
}

// ValidPrincipal reports whether p is a user, a `@group` or `*`.
func ValidPrincipal(p string) bool {
	if p == "*" {
		return true
	}

	if strings.HasPrefix(p, "@") {
		return groupNamePattern.MatchString(strings.TrimPrefix(p, "@"))
	}

	return userNamePattern.MatchString(p)
}
