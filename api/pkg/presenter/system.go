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
	"github.com/mmhk/svn-admin/pkg/view"
)

// System serves the health/readiness/system-info use cases.
type System struct {
	version string
	cfg     config.Config
	now     func() time.Time
}

// NewSystem builds the system presenter.
func NewSystem(version string, cfg config.Config) *System {
	return &System{version: version, cfg: cfg, now: time.Now}
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
func (p *System) Ready(_ context.Context) view.Readiness {
	checks := []view.ReadinessCheck{
		checkDirectory("svnParentPath", p.cfg.SVNParentPath),
		checkFile("svnAuthFile", p.cfg.SVNAuthFile),
		checkFile("svnAccessFile", p.cfg.SVNAccessFile),
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
