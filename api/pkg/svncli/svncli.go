// Package svncli is the thin, argv-based wrapper around the Subversion command
// line tools (plan section 5.4).
//
// Hard rules enforced here:
//   - no shell: every call is argv, never a concatenated command string;
//   - the child process never prompts (`--non-interactive`, stdin is empty);
//   - the locale is pinned to C so revision/date parsing is deterministic;
//   - every invocation has a deadline;
//   - failures become *svncli.Error carrying the exit code and every SVN
//     diagnostic code found on stderr (E160013, but also the W160013 warning
//     the Windows CLI prints first). The raw stderr is written to the service
//     log and is never handed to an HTTP client.
package svncli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// Default timeouts from plan section 5.4: reads are 30s, creating a repository
// (svnadmin create + layout) is allowed 300s.
const (
	DefaultTimeout = 30 * time.Second
	LongTimeout    = 300 * time.Second
)

// svnDiagnostic matches every Subversion diagnostic code such as E160013 or
// W160013. One failure can print several of them: the Windows builds of the
// CLI (SlikSVN) report a missing path as "W160013: ... not found" followed by
// "E200009: ...", so a single code is not enough to classify the failure.
var svnDiagnostic = regexp.MustCompile(`\b[EW]\d{6}\b`)

// notFoundCodes are the diagnostics that mean "this path does not exist in the
// repository": W160013 is the warning spelling, E160013 and E200009 the error
// spellings of the same condition.
var notFoundCodes = map[string]bool{
	"E160013": true,
	"E200009": true,
	"W160013": true,
}

// svnDiagnostics returns every distinct code found in stderr, in the order the
// CLI printed them.
func svnDiagnostics(stderr []byte) []string {
	matches := svnDiagnostic.FindAllString(string(stderr), -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]bool, len(matches))
	codes := make([]string, 0, len(matches))

	for _, code := range matches {
		if seen[code] {
			continue
		}

		seen[code] = true

		codes = append(codes, code)
	}

	return codes
}

// primaryCode prefers the first error code over warnings: an E code explains
// why the command failed, a W code only explains what it noticed on the way.
func primaryCode(codes []string) string {
	for _, code := range codes {
		if strings.HasPrefix(code, "E") {
			return code
		}
	}

	if len(codes) > 0 {
		return codes[0]
	}

	return ""
}

// Runner executes one program with an argv slice. It exists so tests can
// assert the exact argv without a real Subversion installation.
type Runner interface {
	Run(ctx context.Context, name string, args ...string) (stdout, stderr []byte, err error)
}

// ExecRunner is the production Runner: exec.CommandContext without a shell.
type ExecRunner struct{}

var _ Runner = ExecRunner{}

// Run implements Runner.
func (ExecRunner) Run(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
	command := exec.CommandContext(ctx, name, args...)
	command.Env = commandEnv()

	// A nil Stdin means the child gets an empty stdin, so svn can never block
	// on a credential prompt.
	var stdout, stderr bytes.Buffer

	command.Stdout = &stdout
	command.Stderr = &stderr

	err := command.Run()

	return stdout.Bytes(), stderr.Bytes(), err
}

// commandEnv pins the locale without duplicating existing LANG/LC_* entries
// (glibc returns the first match, so appending alone would be a no-op).
func commandEnv() []string {
	environ := os.Environ()
	out := make([]string, 0, len(environ)+2)

	for _, entry := range environ {
		switch {
		case strings.HasPrefix(entry, "LC_ALL="),
			strings.HasPrefix(entry, "LANG="),
			strings.HasPrefix(entry, "LC_MESSAGES="):
			continue
		}

		out = append(out, entry)
	}

	return append(out, "LC_ALL=C", "LANG=C")
}

// Error is the typed failure of one Subversion invocation.
//
// The message intentionally contains no stderr text and no filesystem path:
// stderr is logged by Client, and handlers only ever see this safe summary.
type Error struct {
	Op       string
	Command  string
	ExitCode int
	// Code is the primary diagnostic: the first E series code when the CLI
	// printed one, otherwise the first warning code. Empty when stderr had no
	// SVN code at all.
	Code string
	// Codes holds every distinct diagnostic found in stderr, in order.
	Codes    []string
	TimedOut bool
	Err      error
}

func (e *Error) Error() string {
	switch {
	case e.TimedOut:
		return fmt.Sprintf("%s %s timed out", e.Command, e.Op)
	case e.ExitCode != 0:
		return fmt.Sprintf("%s %s failed with exit code %d", e.Command, e.Op, e.ExitCode)
	case e.Err != nil:
		return fmt.Sprintf("%s %s failed: %v", e.Command, e.Op, e.Err)
	default:
		return fmt.Sprintf("%s %s failed", e.Command, e.Op)
	}
}

// Unwrap keeps errors.Is(err, context.DeadlineExceeded) working.
func (e *Error) Unwrap() error {
	return e.Err
}

// IsNotFound reports whether the command failed because the path does not
// exist in the repository (plan T4.2): SVN E160013 / E200009, or the W160013
// warning that the Windows CLI prints before E200009.
//
// Error codes win over warnings: when stderr carries any E series code, only
// those are evaluated. A W160013 warning must not turn an unrelated E-level
// failure into a 404.
func IsNotFound(err error) bool {
	var target *Error
	if !errors.As(err, &target) {
		return false
	}

	if len(target.Codes) == 0 {
		// Tolerate a hand-built Error that only set Code.
		return notFoundCodes[target.Code]
	}

	sawError := false

	for _, code := range target.Codes {
		if !strings.HasPrefix(code, "E") {
			continue
		}

		sawError = true

		if notFoundCodes[code] {
			return true
		}
	}

	if sawError {
		return false
	}

	for _, code := range target.Codes {
		if notFoundCodes[code] {
			return true
		}
	}

	return false
}

// Options configures a Client.
type Options struct {
	Runner      Runner
	Logger      *slog.Logger
	SVN         string
	SVNAdmin    string
	Timeout     time.Duration
	LongTimeout time.Duration
}

// Client invokes svn/svnadmin.
type Client struct {
	runner      Runner
	logger      *slog.Logger
	svn         string
	svnAdmin    string
	timeout     time.Duration
	longTimeout time.Duration
}

// New builds a Client, filling in the production defaults.
func New(opts Options) *Client {
	if opts.Runner == nil {
		opts.Runner = ExecRunner{}
	}

	if strings.TrimSpace(opts.SVN) == "" {
		opts.SVN = "svn"
	}

	if strings.TrimSpace(opts.SVNAdmin) == "" {
		opts.SVNAdmin = "svnadmin"
	}

	if opts.Timeout <= 0 {
		opts.Timeout = DefaultTimeout
	}

	if opts.LongTimeout <= 0 {
		opts.LongTimeout = LongTimeout
	}

	return &Client{
		runner:      opts.Runner,
		logger:      opts.Logger,
		svn:         opts.SVN,
		svnAdmin:    opts.SVNAdmin,
		timeout:     opts.Timeout,
		longTimeout: opts.LongTimeout,
	}
}

// run executes one command under a deadline and converts every failure into a
// typed *Error. stderr is logged, never returned to the caller.
func (c *Client) run(ctx context.Context, op string, timeout time.Duration, name string, args ...string) ([]byte, error) {
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	stdout, stderr, err := c.runner.Run(callCtx, name, args...)
	if err == nil {
		return stdout, nil
	}

	codes := svnDiagnostics(stderr)

	failure := &Error{
		Op:       op,
		Command:  name,
		ExitCode: exitCode(err),
		Code:     primaryCode(codes),
		Codes:    codes,
		TimedOut: errors.Is(callCtx.Err(), context.DeadlineExceeded),
		Err:      err,
	}

	if c.logger != nil {
		c.logger.Error(
			"svn command failed",
			slog.String("command", name),
			slog.String("op", op),
			slog.Int("exit_code", failure.ExitCode),
			slog.Bool("timed_out", failure.TimedOut),
			slog.String("code", failure.Code),
			slog.String("stderr", truncateForLog(string(stderr))),
		)
	}

	return nil, failure
}

func exitCode(err error) int {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}

	return 0
}

const maxLogStderr = 4096

func truncateForLog(value string) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) <= maxLogStderr {
		return trimmed
	}

	return trimmed[:maxLogStderr] + "…(truncated)"
}
