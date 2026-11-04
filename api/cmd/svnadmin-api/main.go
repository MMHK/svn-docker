// Command svnadmin-api is the SVN administration service.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mmhk/svn-admin/pkg/auth"
	"github.com/mmhk/svn-admin/pkg/config"
	"github.com/mmhk/svn-admin/pkg/presenter"
	"github.com/mmhk/svn-admin/pkg/store/authz"
	"github.com/mmhk/svn-admin/pkg/store/htpasswd"
	"github.com/mmhk/svn-admin/pkg/store/roles"
	"github.com/mmhk/svn-admin/pkg/store/svnrepos"
	"github.com/mmhk/svn-admin/pkg/svncli"
	"github.com/mmhk/svn-admin/pkg/transport/httpapi"
	"github.com/mmhk/svn-admin/pkg/webui"
)

// version is overridden at build time:
//
//	go build -ldflags "-X main.version=3.0.0"
var version = "3.0.0-dev"

// @title						SVN Admin API
// @version					3.0.0-dev
// @description				SVN administration API. The service is reachable only from RFC1918 networks and, from P1, requires a bearer session token.
// @description				Error responses always use {"error":{"code":"...","message":"..."}}.
// @BasePath					/
//
// @securityDefinitions.apikey	bearerAuth
// @in							header
// @name						Authorization
// @description				Bearer session token returned by POST /svnadmin/api/v1/session, e.g. "Bearer svna_...".
func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	command := "serve"
	if len(args) > 0 {
		command = args[0]
	}

	switch command {
	case "serve":
		return serve(stderr)
	case "check":
		return check(stdout, stderr)
	case "version", "-v", "--version":
		fmt.Fprintln(stdout, version)

		return 0
	case "help", "-h", "--help":
		usage(stdout)

		return 0
	default:
		fmt.Fprintf(stderr, "unknown command %q\n\n", command)
		usage(stderr)

		return 2
	}
}

func usage(w io.Writer) {
	fmt.Fprint(w, `svnadmin-api `+version+`

Usage:
  svnadmin-api [serve]   start the HTTP API (default)
  svnadmin-api check     validate config and readiness, then exit
  svnadmin-api version   print the version
`)
}

func serve(stderr io.Writer) int {
	logger := newLogger(stderr).With(slog.String("version", version))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration error", slog.String("error", err.Error()))

		return 1
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	allowlist, err := buildAllowlist(ctx, cfg, logger)
	if err != nil {
		logger.Error("allowlist configuration error", slog.String("error", err.Error()))

		return 1
	}

	web, err := webui.Handler()
	if err != nil {
		logger.Error("embedded web UI is unusable", slog.String("error", err.Error()))

		return 1
	}

	users := htpasswd.New(cfg.SVNAuthFile, htpasswd.WithBackupDir(cfg.BackupDir))
	groupStore := authz.New(cfg.SVNAccessFile, authz.WithBackupDir(cfg.BackupDir))
	roleStore := roles.New(cfg.RoleFile, roles.WithBackupDir(cfg.BackupDir))

	svnCommand := svncli.New(svncli.Options{
		Runner:   svncli.ExecRunner{},
		Logger:   logger,
		SVN:      cfg.SVNBinary,
		SVNAdmin: cfg.SVNAdminBinary,
	})
	repoStore := svnrepos.NewStore(svnrepos.Options{
		Parents:  []string{cfg.SVNParentPath},
		CLI:      svnCommand,
		TrashDir: cfg.TrashDir,
	})

	// Plan T4.5 / D7: deleted repositories stay restorable for the retention
	// period. The sweep runs once at startup and then on the configured
	// interval; it is independent of SVN_ADMIN_REPO_DELETE_ENABLED, because
	// disabling new deletions must not keep old trash forever.
	purger := svnrepos.NewTrashPurger(svnrepos.TrashPurgeOptions{
		Dir:       cfg.TrashDir,
		Retention: cfg.TrashRetention,
		Logger:    logger,
	})
	logger.Info(
		"trash retention sweep scheduled",
		slog.String("dir", cfg.TrashDir),
		slog.String("retention", cfg.TrashRetention.String()),
		slog.String("interval", cfg.TrashPurgeInterval.String()),
	)

	go purger.Run(ctx, cfg.TrashPurgeInterval)

	tokens := auth.NewManager(auth.Options{TTL: cfg.TokenTTL, Idle: cfg.TokenIdle})
	limiter := auth.NewRateLimiter(cfg.LoginRatePerMin, time.Minute)

	bootstrapToken := cfg.BootstrapToken
	if bootstrapToken == "" {
		generated, err := auth.NewBootstrapToken()
		if err != nil {
			logger.Error("cannot generate bootstrap token", slog.String("error", err.Error()))

			return 1
		}

		bootstrapToken = generated

		logger.Warn(
			"no SVN_ADMIN_BOOTSTRAP_TOKEN set; generated a one-time bootstrap token for POST /svnadmin/api/v1/setup/admin",
			slog.String("bootstrap_token", generated),
		)
	}

	handler, err := httpapi.New(httpapi.Options{
		Config:    cfg,
		Logger:    logger,
		Allowlist: allowlist,
		Tokens:    tokens,
		System:    presenter.NewSystem(version, cfg, roleStore),
		Session:   presenter.NewSession(users, roleStore, tokens, limiter),
		Setup: presenter.NewSetup(presenter.SetupDeps{
			Roles:          roleStore,
			Users:          users,
			Accounts:       users,
			AuthFile:       cfg.SVNAuthFile,
			AccessFile:     cfg.SVNAccessFile,
			RoleFile:       cfg.RoleFile,
			BootstrapToken: bootstrapToken,
		}),
		User: presenter.NewUser(presenter.UserDeps{
			Users:       users,
			Accounts:    users,
			Roles:       roleStore,
			RoleWriter:  roleStore,
			Groups:      groupStore,
			Authz:       groupStore,
			AccessPaths: groupStore,
			Tokens:      tokens,
		}),
		Group: presenter.NewGroup(groupStore, groupStore, groupStore),
		Repo: presenter.NewRepository(presenter.RepositoryDeps{
			Repos:         repoStore,
			Writer:        repoStore,
			Browser:       repoStore,
			AccessPaths:   groupStore,
			DeleteEnabled: cfg.RepoDelete,
		}),
		AccessPath: presenter.NewAccessPath(
			groupStore,
			groupStore,
			roleStore,
			roleStore,
		),
		Role: presenter.NewRole(roleStore),
		Web:  web,
	})
	if err != nil {
		logger.Error("router construction failed", slog.String("error", err.Error()))

		return 1
	}

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelWarn),
	}

	serverErr := make(chan error, 1)

	go func() {
		logger.Info(
			"svnadmin-api listening",
			slog.String("addr", cfg.Addr),
			slog.Bool("docs_enabled", cfg.EnableDocs),
			slog.String("default_locale", cfg.DefaultLocale),
		)

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		logger.Error("http server failed", slog.String("error", err.Error()))

		return 1
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", slog.String("error", err.Error()))
		_ = server.Close()

		return 1
	}

	logger.Info("server stopped")

	return 0
}

func check(stdout, stderr io.Writer) int {
	logger := newLogger(stderr)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration error", slog.String("error", err.Error()))

		return 1
	}

	readiness := presenter.NewSystem(version, cfg).Ready(context.Background())

	encoded, err := json.MarshalIndent(readiness, "", "  ")
	if err != nil {
		logger.Error("cannot encode readiness", slog.String("error", err.Error()))

		return 1
	}

	fmt.Fprintln(stdout, string(encoded))

	if readiness.Status != "ok" {
		return 1
	}

	return 0
}

func buildAllowlist(ctx context.Context, cfg config.Config, logger *slog.Logger) (*httpapi.Allowlist, error) {
	resolved, unresolved := httpapi.ResolveTrustedProxies(ctx, cfg.TrustedProxies)

	for _, proxy := range unresolved {
		logger.Warn(
			"trusted proxy could not be resolved; X-Forwarded-For from it will be ignored",
			slog.String("proxy", proxy),
		)
	}

	return httpapi.NewAllowlist(cfg.AllowCIDRs, resolved)
}

func newLogger(w io.Writer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})).With(slog.String("service", "svnadmin-api"))
}
