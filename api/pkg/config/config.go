// Package config loads the svn-admin service settings from the environment.
//
// Every setting uses the SVN_ADMIN_ prefix. The three Subversion paths also
// accept the historical SVN_* spelling as a fallback so existing deployment
// env files keep working during the migration; the prefixed name always wins.
// The legacy data/config.ini is deliberately not read (import happens in P1).
package config

import (
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// EnvPrefix is prepended to every setting name.
const EnvPrefix = "SVN_ADMIN_"

// Supported display locales (plan D15).
const (
	LocaleZhCN = "zh_CN"
	LocaleEnUS = "en_US"
)

// Config is the fully validated service configuration.
type Config struct {
	Addr               string
	SVNParentPath      string
	SVNAuthFile        string
	SVNAccessFile      string
	ConfigINI          string
	BackupDir          string
	TrashDir           string
	SVNBinary          string
	SVNAdminBinary     string
	BootstrapToken     string
	AllowCIDRs         []string
	TrustedProxies     []string
	UID                int
	GID                int
	TokenTTL           time.Duration
	TokenIdle          time.Duration
	TrashRetention     time.Duration
	TrashPurgeInterval time.Duration
	LoginRatePerMin    int
	EnableDocs         bool
	RepoDelete         bool
	DefaultLocale      string
	ShutdownTimeout    time.Duration

	// explicit records which SVN paths came from the environment. The legacy
	// data/config.ini fallback must never override an operator setting.
	explicit pathExplicit
}

type pathExplicit struct {
	parent bool
	auth   bool
	access bool
}

// Load reads the process environment and then applies the read-only
// data/config.ini fallback for any SVN path the environment did not set.
func Load() (Config, error) {
	cfg, err := LoadFrom(os.LookupEnv)
	if err != nil {
		return Config{}, err
	}

	if err := cfg.applyLegacyINI(); err != nil {
		return Config{}, err
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// LoadFrom reads settings through lookup, which makes the defaults and the
// error paths testable without touching the process environment. It does not
// read data/config.ini; Load does that as a best-effort fallback.
func LoadFrom(lookup func(string) (string, bool)) (Config, error) {
	l := loader{lookup: lookup}

	cfg := Config{
		Addr:            l.str("ADDR", ":8080"),
		SVNParentPath:   l.strAlias("SVN_PARENT_PATH", "/var/svnlocal"),
		SVNAuthFile:     l.strAlias("SVN_AUTH_FILE", "/var/svnlocal/svn-auth-file"),
		SVNAccessFile:   l.strAlias("SVN_ACCESS_FILE", "/var/svnlocal/svn-access-file"),
		ConfigINI:       l.str("CONFIG_INI", "data/config.ini"),
		BootstrapToken:  l.str("BOOTSTRAP_TOKEN", ""),
		AllowCIDRs:      l.list("ALLOW_CIDRS", []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"}),
		TrustedProxies:  l.list("TRUSTED_PROXIES", []string{"svn"}),
		DefaultLocale:   l.str("DEFAULT_LOCALE", LocaleZhCN),
		ShutdownTimeout: 10 * time.Second,
	}

	cfg.explicit = pathExplicit{
		parent: l.present("SVN_PARENT_PATH"),
		auth:   l.present("SVN_AUTH_FILE"),
		access: l.present("SVN_ACCESS_FILE"),
	}

	// Backups go to data/backup and deleted repositories to data/trash, which
	// is what the v2 tree already uses (plan section 5.1, D7).
	cfg.BackupDir = l.str("BACKUP_DIR", filepath.Join(filepath.Dir(cfg.ConfigINI), "backup"))
	cfg.TrashDir = l.str("TRASH_DIR", filepath.Join(filepath.Dir(cfg.ConfigINI), "trash"))
	cfg.SVNBinary = l.str("SVN_BINARY", "svn")
	cfg.SVNAdminBinary = l.str("SVNADMIN_BINARY", "svnadmin")

	var err error
	if cfg.UID, err = l.integer("UID", 1000); err != nil {
		return Config{}, err
	}

	if cfg.GID, err = l.integer("GID", 1000); err != nil {
		return Config{}, err
	}

	if cfg.TokenTTL, err = l.duration("TOKEN_TTL", 12*time.Hour); err != nil {
		return Config{}, err
	}

	if cfg.TokenIdle, err = l.duration("TOKEN_IDLE", 2*time.Hour); err != nil {
		return Config{}, err
	}

	// Plan D7 / T4.5: deleted repositories stay restorable for 30 days, and the
	// sweeper runs once a day (plus once at startup).
	if cfg.TrashRetention, err = l.duration("TRASH_RETENTION", 720*time.Hour); err != nil {
		return Config{}, err
	}

	if cfg.TrashPurgeInterval, err = l.duration("TRASH_PURGE_INTERVAL", 24*time.Hour); err != nil {
		return Config{}, err
	}

	if cfg.LoginRatePerMin, err = l.integer("LOGIN_RATE_PER_MIN", 5); err != nil {
		return Config{}, err
	}

	if cfg.EnableDocs, err = l.boolean("ENABLE_DOCS", true); err != nil {
		return Config{}, err
	}

	// Plan D20: repository deletion is enabled by default; the trash rename
	// plus the name confirmation are the safety net.
	if cfg.RepoDelete, err = l.boolean("REPO_DELETE_ENABLED", true); err != nil {
		return Config{}, err
	}

	if cfg.ShutdownTimeout, err = l.duration("SHUTDOWN_TIMEOUT", 10*time.Second); err != nil {
		return Config{}, err
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// Validate enforces the cross-field invariants and normalises the CIDR list.
func (c *Config) Validate() error {
	if strings.TrimSpace(c.Addr) == "" {
		return fmt.Errorf("config: %sADDR must not be empty", EnvPrefix)
	}

	if c.SVNParentPath == "" {
		return fmt.Errorf("config: %sSVN_PARENT_PATH must not be empty", EnvPrefix)
	}

	if c.SVNAuthFile == "" {
		return fmt.Errorf("config: %sSVN_AUTH_FILE must not be empty", EnvPrefix)
	}

	if c.SVNAccessFile == "" {
		return fmt.Errorf("config: %sSVN_ACCESS_FILE must not be empty", EnvPrefix)
	}

	if c.BackupDir == "" {
		return fmt.Errorf("config: %sBACKUP_DIR must not be empty", EnvPrefix)
	}

	if c.TrashDir == "" {
		return fmt.Errorf("config: %sTRASH_DIR must not be empty", EnvPrefix)
	}

	if strings.TrimSpace(c.SVNBinary) == "" {
		return fmt.Errorf("config: %sSVN_BINARY must not be empty", EnvPrefix)
	}

	if strings.TrimSpace(c.SVNAdminBinary) == "" {
		return fmt.Errorf("config: %sSVNADMIN_BINARY must not be empty", EnvPrefix)
	}

	if c.UID < 0 {
		return fmt.Errorf("config: %sUID must not be negative", EnvPrefix)
	}

	if c.GID < 0 {
		return fmt.Errorf("config: %sGID must not be negative", EnvPrefix)
	}

	if c.TokenTTL <= 0 {
		return fmt.Errorf("config: %sTOKEN_TTL must be positive", EnvPrefix)
	}

	if c.TokenIdle <= 0 {
		return fmt.Errorf("config: %sTOKEN_IDLE must be positive", EnvPrefix)
	}

	if c.TokenIdle > c.TokenTTL {
		return fmt.Errorf("config: %sTOKEN_IDLE must not exceed %sTOKEN_TTL", EnvPrefix, EnvPrefix)
	}

	if c.TrashRetention <= 0 {
		return fmt.Errorf("config: %sTRASH_RETENTION must be positive", EnvPrefix)
	}

	if c.TrashPurgeInterval <= 0 {
		return fmt.Errorf("config: %sTRASH_PURGE_INTERVAL must be positive", EnvPrefix)
	}

	if c.LoginRatePerMin <= 0 {
		return fmt.Errorf("config: %sLOGIN_RATE_PER_MIN must be positive", EnvPrefix)
	}

	if c.ShutdownTimeout <= 0 {
		return fmt.Errorf("config: %sSHUTDOWN_TIMEOUT must be positive", EnvPrefix)
	}

	switch c.DefaultLocale {
	case LocaleZhCN, LocaleEnUS:
	default:
		return fmt.Errorf(
			"config: %sDEFAULT_LOCALE=%q is not supported (want %s or %s)",
			EnvPrefix, c.DefaultLocale, LocaleZhCN, LocaleEnUS,
		)
	}

	normalised := make([]string, 0, len(c.AllowCIDRs))
	for _, raw := range c.AllowCIDRs {
		prefix, err := parsePrefix(raw)
		if err != nil {
			return fmt.Errorf("config: %sALLOW_CIDRS=%q: %w", EnvPrefix, raw, err)
		}

		normalised = append(normalised, prefix.String())
	}

	if len(normalised) == 0 {
		return fmt.Errorf("config: %sALLOW_CIDRS must contain at least one network", EnvPrefix)
	}

	c.AllowCIDRs = normalised

	return nil
}

// parsePrefix accepts both CIDR notation and a bare IP address.
func parsePrefix(raw string) (netip.Prefix, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return netip.Prefix{}, fmt.Errorf("empty network")
	}

	if prefix, err := netip.ParsePrefix(value); err == nil {
		return prefix.Masked(), nil
	}

	addr, err := netip.ParseAddr(value)
	if err != nil {
		return netip.Prefix{}, fmt.Errorf("not an IP address or CIDR")
	}

	return netip.PrefixFrom(addr, addr.BitLen()), nil
}

type loader struct {
	lookup func(string) (string, bool)
}

func (l loader) raw(name string) (string, bool) {
	value, ok := l.lookup(EnvPrefix + name)
	if !ok || strings.TrimSpace(value) == "" {
		return "", false
	}

	return strings.TrimSpace(value), true
}

func (l loader) str(name, def string) string {
	if value, ok := l.raw(name); ok {
		return value
	}

	return def
}

// present reports whether the canonical or legacy spelling is set to a
// non-empty value.
func (l loader) present(name string) bool {
	if _, ok := l.raw(name); ok {
		return true
	}

	value, ok := l.lookup(name)

	return ok && strings.TrimSpace(value) != ""
}

// strAlias falls back to the legacy unprefixed spelling when the canonical
// SVN_ADMIN_* variable is absent.
func (l loader) strAlias(name, def string) string {
	if value, ok := l.raw(name); ok {
		return value
	}

	if value, ok := l.lookup(name); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}

	return def
}

func (l loader) list(name string, def []string) []string {
	value, ok := l.raw(name)
	if !ok {
		return def
	}

	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))

	for _, part := range parts {
		if item := strings.TrimSpace(part); item != "" {
			out = append(out, item)
		}
	}

	return out
}

func (l loader) integer(name string, def int) (int, error) {
	value, ok := l.raw(name)
	if !ok {
		return def, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("config: %s%s=%q: %w", EnvPrefix, name, value, err)
	}

	return parsed, nil
}

func (l loader) duration(name string, def time.Duration) (time.Duration, error) {
	value, ok := l.raw(name)
	if !ok {
		return def, nil
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("config: %s%s=%q: %w", EnvPrefix, name, value, err)
	}

	return parsed, nil
}

func (l loader) boolean(name string, def bool) (bool, error) {
	value, ok := l.raw(name)
	if !ok {
		return def, nil
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("config: %s%s=%q: %w", EnvPrefix, name, value, err)
	}

	return parsed, nil
}
