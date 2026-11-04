package config

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func lookupFrom(env map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		value, ok := env[key]
		return value, ok
	}
}

func TestLoadDefaults(t *testing.T) {
	cfg, err := LoadFrom(lookupFrom(nil))
	if err != nil {
		t.Fatalf("LoadFrom() error = %v", err)
	}

	if cfg.Addr != ":8080" {
		t.Errorf("Addr = %q, want :8080", cfg.Addr)
	}

	if cfg.SVNParentPath != "/var/svnlocal" {
		t.Errorf("SVNParentPath = %q", cfg.SVNParentPath)
	}

	if cfg.SVNAuthFile != "/var/svnlocal/svn-auth-file" {
		t.Errorf("SVNAuthFile = %q", cfg.SVNAuthFile)
	}

	if cfg.SVNAccessFile != "/var/svnlocal/svn-access-file" {
		t.Errorf("SVNAccessFile = %q", cfg.SVNAccessFile)
	}

	wantCIDRs := []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"}
	if !reflect.DeepEqual(cfg.AllowCIDRs, wantCIDRs) {
		t.Errorf("AllowCIDRs = %v, want %v", cfg.AllowCIDRs, wantCIDRs)
	}

	if !reflect.DeepEqual(cfg.TrustedProxies, []string{"svn"}) {
		t.Errorf("TrustedProxies = %v, want [svn]", cfg.TrustedProxies)
	}

	if cfg.UID != 1000 || cfg.GID != 1000 {
		t.Errorf("UID/GID = %d/%d, want 1000/1000", cfg.UID, cfg.GID)
	}

	if cfg.TokenTTL != 12*time.Hour {
		t.Errorf("TokenTTL = %v, want 12h", cfg.TokenTTL)
	}

	if cfg.TokenIdle != 2*time.Hour {
		t.Errorf("TokenIdle = %v, want 2h", cfg.TokenIdle)
	}

	if !cfg.EnableDocs {
		t.Error("EnableDocs = false, want true")
	}

	if cfg.DefaultLocale != LocaleZhCN {
		t.Errorf("DefaultLocale = %q, want %q", cfg.DefaultLocale, LocaleZhCN)
	}

	if cfg.ShutdownTimeout != 10*time.Second {
		t.Errorf("ShutdownTimeout = %v, want 10s", cfg.ShutdownTimeout)
	}

	if cfg.TrashDir != filepath.Join("data", "trash") {
		t.Errorf("TrashDir = %q, want data/trash", cfg.TrashDir)
	}

	if cfg.TrashRetention != 720*time.Hour {
		t.Errorf("TrashRetention = %v, want 720h (30 days)", cfg.TrashRetention)
	}

	if cfg.TrashPurgeInterval != 24*time.Hour {
		t.Errorf("TrashPurgeInterval = %v, want 24h", cfg.TrashPurgeInterval)
	}

	if !cfg.RepoDelete {
		t.Error("RepoDelete = false, want true (plan D20)")
	}

	if cfg.SVNBinary != "svn" || cfg.SVNAdminBinary != "svnadmin" {
		t.Errorf("SVN binaries = %q/%q", cfg.SVNBinary, cfg.SVNAdminBinary)
	}
}

func TestLoadOverrides(t *testing.T) {
	cfg, err := LoadFrom(lookupFrom(map[string]string{
		"SVN_ADMIN_ADDR":             "127.0.0.1:9090",
		"SVN_ADMIN_SVN_PARENT_PATH":  "/srv/svn",
		"SVN_ADMIN_SVN_AUTH_FILE":    "/srv/svn/auth",
		"SVN_ADMIN_SVN_ACCESS_FILE":  "/srv/svn/access",
		"SVN_ADMIN_ALLOW_CIDRS":      " 10.1.0.0/16 , 192.168.5.5 ,",
		"SVN_ADMIN_TRUSTED_PROXIES":  "10.0.0.2,proxy.local",
		"SVN_ADMIN_UID":              "1234",
		"SVN_ADMIN_GID":              "5678",
		"SVN_ADMIN_TOKEN_TTL":        "24h",
		"SVN_ADMIN_TOKEN_IDLE":       "3h",
		"SVN_ADMIN_ENABLE_DOCS":      "false",
		"SVN_ADMIN_DEFAULT_LOCALE":   "en_US",
		"SVN_ADMIN_SHUTDOWN_TIMEOUT": "5s",
	}))
	if err != nil {
		t.Fatalf("LoadFrom() error = %v", err)
	}

	if cfg.Addr != "127.0.0.1:9090" {
		t.Errorf("Addr = %q", cfg.Addr)
	}

	if cfg.SVNParentPath != "/srv/svn" || cfg.SVNAuthFile != "/srv/svn/auth" || cfg.SVNAccessFile != "/srv/svn/access" {
		t.Errorf("SVN paths = %q/%q/%q", cfg.SVNParentPath, cfg.SVNAuthFile, cfg.SVNAccessFile)
	}

	wantCIDRs := []string{"10.1.0.0/16", "192.168.5.5/32"}
	if !reflect.DeepEqual(cfg.AllowCIDRs, wantCIDRs) {
		t.Errorf("AllowCIDRs = %v, want %v", cfg.AllowCIDRs, wantCIDRs)
	}

	if !reflect.DeepEqual(cfg.TrustedProxies, []string{"10.0.0.2", "proxy.local"}) {
		t.Errorf("TrustedProxies = %v", cfg.TrustedProxies)
	}

	if cfg.UID != 1234 || cfg.GID != 5678 {
		t.Errorf("UID/GID = %d/%d", cfg.UID, cfg.GID)
	}

	if cfg.TokenTTL != 24*time.Hour || cfg.TokenIdle != 3*time.Hour {
		t.Errorf("Token TTL/Idle = %v/%v", cfg.TokenTTL, cfg.TokenIdle)
	}

	if cfg.EnableDocs {
		t.Error("EnableDocs = true, want false")
	}

	if cfg.DefaultLocale != LocaleEnUS {
		t.Errorf("DefaultLocale = %q", cfg.DefaultLocale)
	}

	if cfg.ShutdownTimeout != 5*time.Second {
		t.Errorf("ShutdownTimeout = %v", cfg.ShutdownTimeout)
	}
}

func TestLoadLegacyPathAliases(t *testing.T) {
	cfg, err := LoadFrom(lookupFrom(map[string]string{
		"SVN_PARENT_PATH": "/legacy/svn",
		"SVN_AUTH_FILE":   "/legacy/auth",
		"SVN_ACCESS_FILE": "/legacy/access",
	}))
	if err != nil {
		t.Fatalf("LoadFrom() error = %v", err)
	}

	if cfg.SVNParentPath != "/legacy/svn" || cfg.SVNAuthFile != "/legacy/auth" || cfg.SVNAccessFile != "/legacy/access" {
		t.Errorf("legacy aliases not applied: %q/%q/%q", cfg.SVNParentPath, cfg.SVNAuthFile, cfg.SVNAccessFile)
	}

	// The prefixed name must win over the legacy alias.
	cfg, err = LoadFrom(lookupFrom(map[string]string{
		"SVN_ADMIN_SVN_PARENT_PATH": "/new/svn",
		"SVN_PARENT_PATH":           "/legacy/svn",
	}))
	if err != nil {
		t.Fatalf("LoadFrom() error = %v", err)
	}

	if cfg.SVNParentPath != "/new/svn" {
		t.Errorf("SVNParentPath = %q, want /new/svn", cfg.SVNParentPath)
	}
}

func TestLoadErrors(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		wantEnv string
	}{
		{
			name:    "invalid duration",
			env:     map[string]string{"SVN_ADMIN_TOKEN_TTL": "soon"},
			wantEnv: "SVN_ADMIN_TOKEN_TTL",
		},
		{
			name:    "invalid bool",
			env:     map[string]string{"SVN_ADMIN_ENABLE_DOCS": "maybe"},
			wantEnv: "SVN_ADMIN_ENABLE_DOCS",
		},
		{
			name:    "invalid int",
			env:     map[string]string{"SVN_ADMIN_UID": "root"},
			wantEnv: "SVN_ADMIN_UID",
		},
		{
			name:    "invalid cidr",
			env:     map[string]string{"SVN_ADMIN_ALLOW_CIDRS": "10.0.0.0/33"},
			wantEnv: "SVN_ADMIN_ALLOW_CIDRS",
		},
		{
			name:    "idle exceeds absolute ttl",
			env:     map[string]string{"SVN_ADMIN_TOKEN_IDLE": "48h"},
			wantEnv: "SVN_ADMIN_TOKEN_IDLE",
		},
		{
			name:    "unsupported locale",
			env:     map[string]string{"SVN_ADMIN_DEFAULT_LOCALE": "fr_FR"},
			wantEnv: "SVN_ADMIN_DEFAULT_LOCALE",
		},
		{
			name:    "negative uid",
			env:     map[string]string{"SVN_ADMIN_UID": "-1"},
			wantEnv: "SVN_ADMIN_UID",
		},
		{
			name:    "zero trash retention",
			env:     map[string]string{"SVN_ADMIN_TRASH_RETENTION": "0h"},
			wantEnv: "SVN_ADMIN_TRASH_RETENTION",
		},
		{
			name:    "zero trash purge interval",
			env:     map[string]string{"SVN_ADMIN_TRASH_PURGE_INTERVAL": "-1h"},
			wantEnv: "SVN_ADMIN_TRASH_PURGE_INTERVAL",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := LoadFrom(lookupFrom(test.env))
			if err == nil {
				t.Fatalf("LoadFrom() error = nil, want error mentioning %s", test.wantEnv)
			}

			if !strings.Contains(err.Error(), test.wantEnv) {
				t.Errorf("error %q does not mention %s", err, test.wantEnv)
			}
		})
	}
}
