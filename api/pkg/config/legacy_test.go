package config

import (
	"os"
	"path/filepath"
	"testing"
)

const legacyINI = `[Common]
FirstStart=0

[Subversion]
SVNAuthFile=/fallback/svn-access-file

[Repositories:svnclient]
SVNParentPath=/fallback/svnlocal

[Users:passwd]
SVNUserFile=/fallback/svn-auth-file
`

func writeLegacyINI(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.ini")
	if err := os.WriteFile(path, []byte(content), 0o640); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	return path
}

func TestLoadFallsBackToLegacyINI(t *testing.T) {
	path := writeLegacyINI(t, legacyINI)

	t.Setenv("SVN_ADMIN_CONFIG_INI", path)
	t.Setenv("SVN_ADMIN_SVN_PARENT_PATH", "")
	t.Setenv("SVN_ADMIN_SVN_AUTH_FILE", "")
	t.Setenv("SVN_ADMIN_SVN_ACCESS_FILE", "")
	t.Setenv("SVN_PARENT_PATH", "")
	t.Setenv("SVN_AUTH_FILE", "")
	t.Setenv("SVN_ACCESS_FILE", "")

	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// v2 config.ini naming is inverted: SVNAuthFile is the authz (access) file
	// and SVNUserFile is the htpasswd file.
	if cfg.SVNParentPath != "/fallback/svnlocal" {
		t.Errorf("SVNParentPath = %q", cfg.SVNParentPath)
	}

	if cfg.SVNAccessFile != "/fallback/svn-access-file" {
		t.Errorf("SVNAccessFile = %q", cfg.SVNAccessFile)
	}

	if cfg.SVNAuthFile != "/fallback/svn-auth-file" {
		t.Errorf("SVNAuthFile = %q", cfg.SVNAuthFile)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if string(before) != string(after) {
		t.Error("Load() modified data/config.ini; the fallback must be read-only")
	}
}

func TestLoadEnvironmentWinsOverLegacyINI(t *testing.T) {
	path := writeLegacyINI(t, legacyINI)

	t.Setenv("SVN_ADMIN_CONFIG_INI", path)
	t.Setenv("SVN_ADMIN_SVN_PARENT_PATH", "/env/svnlocal")
	t.Setenv("SVN_ADMIN_SVN_AUTH_FILE", "/env/svn-auth-file")
	t.Setenv("SVN_ADMIN_SVN_ACCESS_FILE", "/env/svn-access-file")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.SVNParentPath != "/env/svnlocal" {
		t.Errorf("SVNParentPath = %q", cfg.SVNParentPath)
	}

	if cfg.SVNAuthFile != "/env/svn-auth-file" || cfg.SVNAccessFile != "/env/svn-access-file" {
		t.Errorf("SVN files = %q/%q", cfg.SVNAuthFile, cfg.SVNAccessFile)
	}
}

func TestLoadWithoutLegacyINIKeepsDefaults(t *testing.T) {
	t.Setenv("SVN_ADMIN_CONFIG_INI", filepath.Join(t.TempDir(), "missing.ini"))
	t.Setenv("SVN_ADMIN_SVN_PARENT_PATH", "")
	t.Setenv("SVN_ADMIN_SVN_AUTH_FILE", "")
	t.Setenv("SVN_ADMIN_SVN_ACCESS_FILE", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.SVNParentPath != "/var/svnlocal" {
		t.Errorf("SVNParentPath = %q, want the built-in default", cfg.SVNParentPath)
	}
}
