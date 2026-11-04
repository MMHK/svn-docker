package model

import (
	"errors"
	"fmt"
	"strings"
)

// OperationError reports a multi-step write that failed partway through, for
// example "user.delete" which has to touch the authz file, the role file and
// the htpasswd file (plan T2.3). P2 uses sequential per-file writes instead of
// a cross-file two-phase commit, so the caller must know which steps already
// happened; the message is safe to return to a client because it only carries
// step names, never a path, hash or password.
type OperationError struct {
	Operation string
	Step      string
	Completed []string
	Err       error
}

func (e *OperationError) Error() string {
	return fmt.Sprintf(
		"%s failed at step %q (completed: %s): %v",
		e.Operation, e.Step, strings.Join(e.Completed, ", "), e.Err,
	)
}

// Unwrap keeps errors.Is / errors.As working for the underlying cause.
func (e *OperationError) Unwrap() error {
	return e.Err
}

// SafeMessage renders the operation, the failed step and the completed steps.
// It deliberately omits Err, which may mention file paths.
func (e *OperationError) SafeMessage() string {
	completed := "none"
	if len(e.Completed) > 0 {
		completed = strings.Join(e.Completed, ", ")
	}

	return fmt.Sprintf(
		"%s failed at step %q; completed steps: %s; the data may be partially updated",
		e.Operation, e.Step, completed,
	)
}

// IsOperation reports whether err wraps a multi-step failure.
func IsOperation(err error) bool {
	var target *OperationError

	return errors.As(err, &target)
}
