// Package presenter orchestrates use cases. It must not import net/http:
// HTTP binding and rendering live in pkg/transport/httpapi.
package presenter

import (
	"context"
	"io"
	"os"
	"runtime"
	"time"

	"github.com/mmhk/svn-admin/pkg/config"
	"github.com/mmhk/svn-admin/pkg/model"
	"github.com/mmhk/svn-admin/pkg/view"
)

// System serves the health/readiness/system-info use cases.
type System struct {
	version string
	cfg     config.Config
	roles   model.RoleStore
	now     func() time.Time
}

// NewSystem builds the system presenter. The optional role store enables the
// administrator readiness probe; without it the probe reports the bootstrap
// state as unknown.
func NewSystem(version string, cfg config.Config, roles ...model.RoleStore) *System {
	system := &System{version: version, cfg: cfg, now: time.Now}

	if len(roles) > 0 {
		system.roles = roles[0]
	}

	return system
}

// Info reports the service version, runtime and the state of the configured
// paths. It never returns file contents.
func (p *System) Info(_ context.Context) view.SystemInfo {
	return view.SystemInfo{
		Version:       p.version,
		GoVersion:     runtime.Version(),
		Platform:      runtime.GOOS + "/" + runtime.GOARCH,
		SVNParentPath: inspectPath(p.cfg.SVNParentPath),
		SVNAuthFile:   inspectPath(p.cfg.SVNAuthFile),
		SVNAccessFile: inspectPath(p.cfg.SVNAccessFile),
		DocsEnabled:   p.cfg.EnableDocs,
		DefaultLocale: p.cfg.DefaultLocale,
		ServerTime:    p.now().UTC(),
	}
}

// Ready checks that the SVN data mount is usable. Messages intentionally do
// not contain absolute paths; only /system/info reveals them.
func (p *System) Ready(ctx context.Context) view.Readiness {
	checks := []view.ReadinessCheck{
		checkDirectory("svnParentPath", p.cfg.SVNParentPath),
		checkFile("svnAuthFile", p.cfg.SVNAuthFile),
		checkFile("svnAccessFile", p.cfg.SVNAccessFile),
		checkSetupFile("roleFile", p.cfg.RoleFile),
		p.checkAdministrator(ctx),
	}

	status := "ok"
	for _, check := range checks {
		if !check.OK {
			status = "unavailable"

			break
		}
	}

	return view.Readiness{Status: status, Checks: checks}
}

// checkSetupFile reports a file the bootstrap flow may create. A missing file
// is not a readiness failure; an unreadable one is.
func checkSetupFile(name, path string) view.ReadinessCheck {
	info := inspectPath(path)
	check := view.ReadinessCheck{Name: name, OK: true}

	switch {
	case !info.Exists:
		check.Message = "missing (setup will create it)"
	case info.IsDir:
		check.OK, check.Message = false, "is a directory"
	case !info.Readable:
		check.OK, check.Message = false, "not readable"
	}

	return check
}

// checkAdministrator reports whether an administrator exists. Missing is a
// normal first-run state, so it stays OK with a hint; an unreadable role file
// is a real failure.
func (p *System) checkAdministrator(ctx context.Context) view.ReadinessCheck {
	check := view.ReadinessCheck{Name: "administrator", OK: true}

	if p.roles == nil {
		check.Message = "missing (bootstrap required)"

		return check
	}

	assignments, err := p.roles.Assignments(ctx)
	if err != nil {
		check.OK, check.Message = false, "not readable"

		return check
	}

	for _, roles := range assignments.Roles {
		for _, role := range roles {
			if role == model.RoleAdministrator {
				return check
			}
		}
	}

	check.Message = "missing (bootstrap required)"

	return check
}

func checkDirectory(name, path string) view.ReadinessCheck {
	info := inspectPath(path)
	check := view.ReadinessCheck{Name: name, OK: true}

	switch {
	case !info.Exists:
		check.OK, check.Message = false, "missing"
	case !info.IsDir:
		check.OK, check.Message = false, "not a directory"
	case !info.Readable:
		check.OK, check.Message = false, "not readable"
	}

	return check
}

func checkFile(name, path string) view.ReadinessCheck {
	info := inspectPath(path)
	check := view.ReadinessCheck{Name: name, OK: true}

	switch {
	case !info.Exists:
		check.OK, check.Message = false, "missing"
	case info.IsDir:
		check.OK, check.Message = false, "is a directory"
	case !info.Readable:
		check.OK, check.Message = false, "not readable"
	}

	return check
}

// inspectPath stat()s a path and probes readability/writability without
// reading or truncating any content.
func inspectPath(path string) view.PathInfo {
	out := view.PathInfo{Path: path}

	stat, err := os.Stat(path)
	if err != nil {
		return out
	}

	out.Exists = true
	out.IsDir = stat.IsDir()
	out.Readable = readable(path, stat.IsDir())
	out.Writable = writable(path, stat.IsDir())

	return out
}

func readable(path string, isDir bool) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	if !isDir {
		return true
	}

	// Opening a directory proves search permission; listing proves read.
	if _, err := file.Readdirnames(1); err != nil && err != io.EOF {
		return false
	}

	return true
}

func writable(path string, isDir bool) bool {
	if !isDir {
		file, err := os.OpenFile(path, os.O_WRONLY, 0)
		if err != nil {
			return false
		}

		_ = file.Close()

		return true
	}

	// O_WRONLY is not valid for directories, so probe with a temporary file
	// and remove it immediately.
	probe, err := os.CreateTemp(path, ".svnadmin-write-probe-*")
	if err != nil {
		return false
	}

	name := probe.Name()
	_ = probe.Close()
	_ = os.Remove(name)

	return true
}
