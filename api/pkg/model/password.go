package model

import (
	"fmt"
	"strings"
)

// MaxPasswordLength bounds a submitted password. APR1 only consumes the first
// 8 bytes of salt and any length of password, but an unbounded value would
// turn every hash into an expensive operation.
const MaxPasswordLength = 256

// ValidateUserName enforces the account naming rule shared by htpasswd and the
// authz file: no whitespace, no ':', no newline, at most 64 characters.
func ValidateUserName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return &ValidationError{Field: "name", Reason: "must not be empty"}
	}

	if !userNamePattern.MatchString(trimmed) {
		return &ValidationError{
			Field:  "name",
			Reason: "must start with a letter or digit and contain only letters, digits, '.', '_' or '-' (max 64)",
		}
	}

	return nil
}

// ValidateGroupName enforces the authz group naming rule.
func ValidateGroupName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return &ValidationError{Field: "group", Reason: "must not be empty"}
	}

	if !groupNamePattern.MatchString(trimmed) {
		return &ValidationError{
			Field:  "group",
			Reason: "must contain only letters, digits, '.', '_' or '-' (max 64)",
		}
	}

	return nil
}

// ValidatePassword enforces the password invariants that do not need a store.
// A password is never echoed back and never logged.
func ValidatePassword(password string) error {
	if password == "" {
		return &ValidationError{Field: "password", Reason: "must not be empty"}
	}

	if len(password) > MaxPasswordLength {
		return &ValidationError{
			Field:  "password",
			Reason: fmt.Sprintf("must be at most %d bytes", MaxPasswordLength),
		}
	}

	for _, r := range password {
		switch r {
		case '\n', '\r', 0:
			return &ValidationError{
				Field:  "password",
				Reason: "must not contain a newline or NUL byte",
			}
		}
	}

	return nil
}
