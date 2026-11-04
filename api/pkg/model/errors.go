// Package model holds the domain entities, their invariants and the store
// ports. It must not import net/http or any transport concern.
package model

import (
	"errors"
	"fmt"
	"time"
)

// Kind names a domain object type. It is used by typed errors so the
// transport layer can render a response without knowing every entity struct.
type Kind string

const (
	KindUser       Kind = "user"
	KindGroup      Kind = "group"
	KindRepository Kind = "repository"
	KindAccessPath Kind = "access-path"
	KindRole       Kind = "role"
	KindSession    Kind = "session"
)

// ErrInvalidCredentials is returned by a credential store when the user is
// unknown or the password does not verify. Callers must not distinguish the
// two cases in their response.
var ErrInvalidCredentials = errors.New("invalid credentials")

// ErrReadOnly is returned by stores whose write path is not implemented yet.
// P1 is read-only; the write use cases arrive in P2/P3.
var ErrReadOnly = errors.New("write operations are not available yet")

// ValidationError reports a malformed or disallowed input value.
type ValidationError struct {
	Field  string
	Reason string
}

func (e *ValidationError) Error() string {
	if e.Field == "" {
		return "validation failed: " + e.Reason
	}

	return fmt.Sprintf("validation failed for %s: %s", e.Field, e.Reason)
}

// NotFoundError reports a missing domain object.
type NotFoundError struct {
	Kind Kind
	Name string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s %q not found", e.Kind, e.Name)
}

// ConflictError reports a concurrent modification or a duplicate name.
type ConflictError struct {
	Kind   Kind
	Name   string
	Reason string
}

func (e *ConflictError) Error() string {
	if e.Reason == "" {
		return fmt.Sprintf("%s %q conflicts with the current state", e.Kind, e.Name)
	}

	return fmt.Sprintf("%s %q conflicts: %s", e.Kind, e.Name, e.Reason)
}

// ForbiddenError reports a denied operation or a rejected client address.
type ForbiddenError struct {
	Reason string
}

func (e *ForbiddenError) Error() string {
	if e.Reason == "" {
		return "forbidden"
	}

	return "forbidden: " + e.Reason
}

// UnauthorizedError reports a missing, malformed, expired or revoked token, or
// failed credentials. The transport maps it to 401.
type UnauthorizedError struct {
	Reason string
}

func (e *UnauthorizedError) Error() string {
	if e.Reason == "" {
		return "unauthorized"
	}

	return "unauthorized: " + e.Reason
}

// RateLimitError reports a login attempt above the configured rate.
type RateLimitError struct {
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("too many login attempts; retry in %s", e.RetryAfter.Round(time.Second))
}

// SetupRequiredError reports that no administrator exists yet. The transport
// maps it to 503 with code setup_required.
type SetupRequiredError struct{}

func (e *SetupRequiredError) Error() string {
	return "administrator setup is required"
}

// IsValidation reports whether err is a validation error.
func IsValidation(err error) bool {
	var target *ValidationError

	return errors.As(err, &target)
}

// IsNotFound reports whether err is a not-found error.
func IsNotFound(err error) bool {
	var target *NotFoundError

	return errors.As(err, &target)
}

// IsConflict reports whether err is a conflict error.
func IsConflict(err error) bool {
	var target *ConflictError

	return errors.As(err, &target)
}

// IsForbidden reports whether err is a forbidden error.
func IsForbidden(err error) bool {
	var target *ForbiddenError

	return errors.As(err, &target)
}

// IsUnauthorized reports whether err is an unauthorized error.
func IsUnauthorized(err error) bool {
	var target *UnauthorizedError

	return errors.As(err, &target) || errors.Is(err, ErrInvalidCredentials)
}

// IsRateLimited reports whether err is a rate-limit error.
func IsRateLimited(err error) bool {
	var target *RateLimitError

	return errors.As(err, &target)
}

// IsSetupRequired reports whether err is a setup-required error.
func IsSetupRequired(err error) bool {
	var target *SetupRequiredError

	return errors.As(err, &target)
}
