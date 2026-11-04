package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/mmhk/svn-admin/pkg/model"
	"github.com/mmhk/svn-admin/pkg/view"
)

const contentTypeJSON = "application/json; charset=utf-8"

// statusRecorder captures the status code and size for the access log.
type statusRecorder struct {
	http.ResponseWriter
	status      int
	bytes       int
	wroteHeader bool
}

func (r *statusRecorder) WriteHeader(code int) {
	if r.wroteHeader {
		return
	}

	r.status = code
	r.wroteHeader = true
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(data []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}

	written, err := r.ResponseWriter.Write(data)
	r.bytes += written

	return written, err
}

// Unwrap lets http.ResponseController reach the underlying writer.
func (r *statusRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", contentTypeJSON)
	w.WriteHeader(status)

	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(true)
	_ = encoder.Encode(payload)
}

func writeEnvelope(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, view.ErrorEnvelope{
		Error: view.ErrorDetail{Code: code, Message: message},
	})
}

// writeError maps a domain error to the shared error envelope. Unexpected
// errors are logged with the request id and never leak their text to clients.
func (a *API) writeError(w http.ResponseWriter, r *http.Request, err error) {
	status, code, message := classifyError(err)

	if status >= http.StatusInternalServerError {
		a.logger.Error(
			"request failed",
			slog.String("error", err.Error()),
			slog.String("request_id", requestIDFromContext(r.Context())),
			slog.String("path", r.URL.Path),
		)
	}

	var rateLimited *model.RateLimitError
	if errors.As(err, &rateLimited) {
		w.Header().Set("Retry-After", strconv.Itoa(int(rateLimited.RetryAfter.Seconds()+0.999)))
	}

	writeEnvelope(w, status, code, message)
}

func classifyError(err error) (status int, code, message string) {
	var (
		validation    *model.ValidationError
		notFound      *model.NotFoundError
		conflict      *model.ConflictError
		forbidden     *model.ForbiddenError
		unauthorized  *model.UnauthorizedError
		rateLimited   *model.RateLimitError
		setupRequired *model.SetupRequiredError
		operation     *model.OperationError
	)

	switch {
	case errors.As(err, &validation):
		return http.StatusUnprocessableEntity, "validation_error", validation.Error()
	case errors.As(err, &notFound):
		return http.StatusNotFound, "not_found", notFound.Error()
	case errors.As(err, &conflict):
		return http.StatusConflict, "conflict", conflict.Error()
	case errors.Is(err, model.ErrReadOnly):
		// The deployment wired a read-only store: say so instead of blaming
		// the request (P4 returns 501 when svncli is not configured).
		return http.StatusNotImplemented, "not_implemented", "this operation is not available in this deployment"
	case errors.As(err, &forbidden):
		return http.StatusForbidden, "forbidden", forbidden.Error()
	case errors.As(err, &operation):
		// Multi-step writes report the failed step and what already ran. The
		// message never carries a path, hash or password.
		return http.StatusInternalServerError, "operation_failed", operation.SafeMessage()
	case errors.As(err, &unauthorized), errors.Is(err, model.ErrInvalidCredentials):
		return http.StatusUnauthorized, "unauthorized", "authentication is required or the credentials are invalid"
	case errors.As(err, &rateLimited):
		return http.StatusTooManyRequests, "rate_limited", "too many login attempts"
	case errors.As(err, &setupRequired):
		return http.StatusServiceUnavailable, "setup_required", "an administrator must be created first"
	default:
		return http.StatusInternalServerError, "internal_error", "internal server error"
	}
}
