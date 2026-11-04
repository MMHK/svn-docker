package svnrepos

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Deleted repositories are parked as `<repo>-<UTC20060102T150405Z>`; a second
// deletion inside the same second appends `-2`, `-3`, ... (see trashTarget).
// The pattern is deliberately strict: anything else in the trash directory is
// left alone, because it was not created by this service.
var trashEntryPattern = regexp.MustCompile(`^([A-Za-z0-9_-]{1,64})-(\d{8}T\d{6}Z)(?:-\d{1,4})?$`)

// trashTimeLayout matches the timestamp component of a trash entry name.
const trashTimeLayout = "20060102T150405Z"

// TrashPurgeOptions configures the retention sweeper (plan T4.5 / D7).
type TrashPurgeOptions struct {
	// Dir is SVN_ADMIN_TRASH_DIR. No path outside it is ever removed.
	Dir string
	// Retention is SVN_ADMIN_TRASH_RETENTION (default 720h = 30 days).
	Retention time.Duration
	// Logger receives one line per deletion. Nil disables logging.
	Logger *slog.Logger
	// Now is injectable so the sweep is deterministic in tests.
	Now func() time.Time
}

// TrashPurgeReport summarises one sweep.
type TrashPurgeReport struct {
	// Removed lists the entry names that were deleted.
	Removed []string
	// Skipped lists the entries that were left alone (unrecognised name,
	// symbolic link, not a directory) or that could not be removed.
	Skipped []string
}

// TrashPurger removes trash entries older than the retention period.
//
// Safety rules, in order:
//   - only direct children of Dir are considered;
//   - only names produced by trashTarget (`<repo>-<ts>` with an optional -N)
//     are eligible, so an operator's own files are never touched;
//   - symbolic links are refused, so a link cannot aim RemoveAll elsewhere;
//   - the trash directory itself is never removed, and a non-positive
//     retention refuses to run instead of deleting everything.
type TrashPurger struct {
	dir       string
	retention time.Duration
	logger    *slog.Logger
	now       func() time.Time
}

// NewTrashPurger builds a purger. A blank directory disables it.
func NewTrashPurger(opts TrashPurgeOptions) *TrashPurger {
	purger := &TrashPurger{
		dir:       strings.TrimSpace(opts.Dir),
		retention: opts.Retention,
		logger:    opts.Logger,
		now:       opts.Now,
	}

	if purger.now == nil {
		purger.now = time.Now
	}

	return purger
}

// Purge deletes every expired entry and reports what it did. A missing trash
// directory is not an error: nothing has been deleted yet.
func (p *TrashPurger) Purge(ctx context.Context) (TrashPurgeReport, error) {
	report := TrashPurgeReport{}

	if p.dir == "" {
		return report, errors.New("svnrepos: the trash directory is not configured")
	}

	if p.retention <= 0 {
		return report, errors.New("svnrepos: the trash retention must be positive")
	}

	entries, err := os.ReadDir(p.dir)
	if errors.Is(err, fs.ErrNotExist) {
		return report, nil
	}

	if err != nil {
		return report, &Error{Op: "purge", Path: p.dir, Err: err}
	}

	root := filepath.Clean(p.dir)
	now := p.now()

	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return report, err
		}

		name := entry.Name()

		match := trashEntryPattern.FindStringSubmatch(name)
		if match == nil {
			report.Skipped = append(report.Skipped, name)
			p.logSkip(name, "unrecognised trash entry name")

			continue
		}

		path := filepath.Join(root, name)

		// Defence in depth: ReadDir already returns direct children, but the
		// join must never resolve outside the trash directory.
		if filepath.Dir(path) != root || path == root {
			report.Skipped = append(report.Skipped, name)
			p.logSkip(name, "not a direct child of the trash directory")

			continue
		}

		info, err := os.Lstat(path)
		if err != nil {
			report.Skipped = append(report.Skipped, name)
			p.logSkip(name, "cannot stat entry")

			continue
		}

		if info.Mode()&os.ModeSymlink != 0 {
			report.Skipped = append(report.Skipped, name)
			p.logSkip(name, "symbolic link")

			continue
		}

		if !info.IsDir() {
			report.Skipped = append(report.Skipped, name)
			p.logSkip(name, "not a directory")

			continue
		}

		deletedAt, err := time.Parse(trashTimeLayout, match[2])
		if err != nil {
			report.Skipped = append(report.Skipped, name)
			p.logSkip(name, "unparseable timestamp")

			continue
		}

		// Either signal may mark the entry as old: the timestamp in its name
		// (the deletion time) or its mtime (a manual Chtimes/restore). The
		// older of the two wins, so an operator can force expiry by touching
		// the mtime, and a copied entry cannot hide its age.
		reference := deletedAt
		if info.ModTime().Before(reference) {
			reference = info.ModTime()
		}

		age := now.Sub(reference)
		if age < p.retention {
			continue
		}

		if err := os.RemoveAll(path); err != nil {
			report.Skipped = append(report.Skipped, name)
			p.logSkip(name, "remove failed: "+err.Error())

			continue
		}

		report.Removed = append(report.Removed, name)

		if p.logger != nil {
			p.logger.Info(
				"trash entry purged",
				slog.String("name", name),
				slog.String("age", age.Truncate(time.Second).String()),
				slog.String("deleted_at", deletedAt.UTC().Format(time.RFC3339)),
				slog.String("retention", p.retention.String()),
			)
		}
	}

	return report, nil
}

// Run purges once at startup and then on every interval tick until ctx ends.
// It never returns an error: a failed sweep is logged and retried on the next
// tick, because a transient permission problem must not stop the service.
func (p *TrashPurger) Run(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 24 * time.Hour
	}

	p.purgeAndLog(ctx, "startup")

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.purgeAndLog(ctx, "scheduled")
		}
	}
}

func (p *TrashPurger) purgeAndLog(ctx context.Context, trigger string) {
	report, err := p.Purge(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return
		}

		if p.logger != nil {
			p.logger.Error(
				"trash purge failed",
				slog.String("trigger", trigger),
				slog.String("dir", p.dir),
				slog.String("error", err.Error()),
			)
		}

		return
	}

	if p.logger != nil && (len(report.Removed) > 0 || len(report.Skipped) > 0) {
		p.logger.Info(
			"trash purge finished",
			slog.String("trigger", trigger),
			slog.Int("removed", len(report.Removed)),
			slog.Int("skipped", len(report.Skipped)),
			slog.String("retention", p.retention.String()),
		)
	}
}

func (p *TrashPurger) logSkip(name, reason string) {
	if p.logger == nil {
		return
	}

	p.logger.Warn(
		"trash entry left in place",
		slog.String("name", name),
		slog.String("reason", reason),
	)
}
