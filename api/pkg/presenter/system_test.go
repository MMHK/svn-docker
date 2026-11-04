package presenter

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mmhk/svn-admin/pkg/config"
)

func testSystem(t *testing.T) (*System, string) {
	t.Helper()

	dir := t.TempDir()
	parent := filepath.Join(dir, "svn")

	if err := os.MkdirAll(parent, 0o755); err != nil {
		t.Fatalf("mkdir parent: %v", err)
	}

	authFile := filepath.Join(dir, "svn-auth-file")
	if err := os.WriteFile(authFile, []byte("admin:$apr1$hash\n"), 0o600); err != nil {
		t.Fatalf("write auth file: %v", err)
	}

	accessFile := filepath.Join(dir, "svn-access-file")
	if err := os.WriteFile(accessFile, []byte("[project:/]\n@devs=rw\n"), 0o600); err != nil {
		t.Fatalf("write access file: %v", err)
	}

	cfg := config.Config{
		SVNParentPath: parent,
		SVNAuthFile:   authFile,
		SVNAccessFile: accessFile,
		EnableDocs:    true,
		DefaultLocale: config.LocaleZhCN,
	}

	return NewSystem("3.0.0-test", cfg), dir
}

func TestSystemInfoReportsPaths(t *testing.T) {
	system, _ := testSystem(t)

	info := system.Info(context.Background())

	if info.Version != "3.0.0-test" {
		t.Errorf("Version = %q", info.Version)
	}

	if info.GoVersion == "" {
		t.Error("GoVersion is empty")
	}

	if info.Platform == "" {
		t.Error("Platform is empty")
	}

	if !info.SVNParentPath.Exists || !info.SVNParentPath.IsDir || !info.SVNParentPath.Readable {
		t.Errorf("SVNParentPath = %+v, want existing readable directory", info.SVNParentPath)
	}

	if !info.SVNAuthFile.Exists || !info.SVNAuthFile.Readable || !info.SVNAuthFile.Writable {
		t.Errorf("SVNAuthFile = %+v, want existing readable+writable file", info.SVNAuthFile)
	}

	if !info.SVNAccessFile.Exists || info.SVNAccessFile.IsDir {
		t.Errorf("SVNAccessFile = %+v", info.SVNAccessFile)
	}

	if !info.DocsEnabled || info.DefaultLocale != config.LocaleZhCN {
		t.Errorf("docs/locale = %v/%q", info.DocsEnabled, info.DefaultLocale)
	}

	if info.ServerTime.IsZero() {
		t.Error("ServerTime is zero")
	}
}

func TestSystemReadyOK(t *testing.T) {
	system, _ := testSystem(t)

	ready := system.Ready(context.Background())

	if ready.Status != "ok" {
		t.Fatalf("Status = %q, checks = %+v", ready.Status, ready.Checks)
	}

	for _, check := range ready.Checks {
		if !check.OK {
			t.Errorf("check %s failed: %s", check.Name, check.Message)
		}
	}
}

func TestSystemReadyReportsMissingFiles(t *testing.T) {
	system, dir := testSystem(t)

	cfg := system.cfg
	if err := os.Remove(cfg.SVNAuthFile); err != nil {
		t.Fatalf("remove auth file: %v", err)
	}

	ready := system.Ready(context.Background())

	if ready.Status != "unavailable" {
		t.Fatalf("Status = %q, want unavailable", ready.Status)
	}

	found := false

	for _, check := range ready.Checks {
		if check.Name == "svnAuthFile" {
			found = true

			if check.OK {
				t.Error("svnAuthFile check is OK after removal")
			}

			if check.Message != "missing" {
				t.Errorf("svnAuthFile message = %q, want missing", check.Message)
			}
		}

		if strings.Contains(check.Message, dir) {
			t.Errorf("check message leaks absolute path: %q", check.Message)
		}
	}

	if !found {
		t.Error("svnAuthFile check is missing from readiness output")
	}
}
