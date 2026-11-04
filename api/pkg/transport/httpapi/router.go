// Package httpapi is the thin HTTP transport: routing, middleware, binding and
// rendering. Business decisions live in pkg/presenter.
package httpapi

import (
	"errors"
	"log/slog"
	"net/http"

	httpSwagger "github.com/swaggo/http-swagger/v2"

	"github.com/mmhk/svn-admin/pkg/auth"
	"github.com/mmhk/svn-admin/pkg/config"
	"github.com/mmhk/svn-admin/pkg/presenter"
)

const (
	apiPrefix  = "/svnadmin/api/"
	docsPrefix = "/svnadmin/api/docs/"
)

// Options are the hand-wired dependencies of the router.
type Options struct {
	Config     config.Config
	Logger     *slog.Logger
	Allowlist  *Allowlist
	Tokens     *auth.Manager
	System     *presenter.System
	Session    *presenter.Session
	Setup      *presenter.Setup
	User       *presenter.User
	Group      *presenter.Group
	Repo       *presenter.Repository
	AccessPath *presenter.AccessPath
	Role       *presenter.Role
	Web        http.Handler
}

// API holds the transport dependencies.
type API struct {
	cfg        config.Config
	logger     *slog.Logger
	allowlist  *Allowlist
	tokens     *auth.Manager
	system     *presenter.System
	session    *presenter.Session
	setup      *presenter.Setup
	user       *presenter.User
	group      *presenter.Group
	repo       *presenter.Repository
	accessPath *presenter.AccessPath
	role       *presenter.Role
	web        http.Handler
}

// New builds the HTTP handler tree.
func New(opts Options) (http.Handler, error) {
	if opts.Logger == nil {
		return nil, errors.New("httpapi: logger is required")
	}

	if opts.Allowlist == nil {
		return nil, errors.New("httpapi: allowlist is required")
	}

	if opts.Tokens == nil {
		return nil, errors.New("httpapi: token manager is required")
	}

	if opts.System == nil {
		return nil, errors.New("httpapi: system presenter is required")
	}

	if opts.Session == nil {
		return nil, errors.New("httpapi: session presenter is required")
	}

	if opts.Setup == nil {
		return nil, errors.New("httpapi: setup presenter is required")
	}

	if opts.User == nil || opts.Group == nil || opts.Repo == nil || opts.AccessPath == nil || opts.Role == nil {
		return nil, errors.New("httpapi: read presenters are required")
	}

	web := opts.Web
	if web == nil {
		web = http.NotFoundHandler()
	}

	api := &API{
		cfg:        opts.Config,
		logger:     opts.Logger,
		allowlist:  opts.Allowlist,
		tokens:     opts.Tokens,
		system:     opts.System,
		session:    opts.Session,
		setup:      opts.Setup,
		user:       opts.User,
		group:      opts.Group,
		repo:       opts.Repo,
		accessPath: opts.AccessPath,
		role:       opts.Role,
		web:        web,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", api.healthz)
	mux.HandleFunc("GET /readyz", api.readyz)

	mux.HandleFunc("POST /svnadmin/api/v1/session", api.login)
	mux.HandleFunc("GET /svnadmin/api/v1/session", api.currentSession)
	mux.HandleFunc("POST /svnadmin/api/v1/session/refresh", api.refreshSession)
	mux.HandleFunc("DELETE /svnadmin/api/v1/session", api.logout)
	mux.HandleFunc("POST /svnadmin/api/v1/setup/admin", api.setupAdmin)

	mux.HandleFunc("GET /svnadmin/api/v1/system/info", api.systemInfo)
	mux.HandleFunc("GET /svnadmin/api/v1/users", api.listUsers)
	mux.HandleFunc("POST /svnadmin/api/v1/users", api.createUser)
	mux.HandleFunc("GET /svnadmin/api/v1/users/{name}", api.getUser)
	mux.HandleFunc("DELETE /svnadmin/api/v1/users/{name}", api.deleteUser)
	mux.HandleFunc("PUT /svnadmin/api/v1/users/{name}/password", api.changeUserPassword)
	mux.HandleFunc("GET /svnadmin/api/v1/users/{name}/roles", api.getUserRoles)
	mux.HandleFunc("PUT /svnadmin/api/v1/users/{name}/roles", api.setUserRoles)
	mux.HandleFunc("GET /svnadmin/api/v1/users/{name}/groups", api.getUserGroups)
	mux.HandleFunc("PUT /svnadmin/api/v1/users/{name}/groups", api.setUserGroups)
	mux.HandleFunc("GET /svnadmin/api/v1/users/{name}/access-paths", api.getUserAccessPaths)

	mux.HandleFunc("GET /svnadmin/api/v1/groups", api.listGroups)
	mux.HandleFunc("POST /svnadmin/api/v1/groups", api.createGroup)
	mux.HandleFunc("GET /svnadmin/api/v1/groups/{name}", api.getGroup)
	mux.HandleFunc("DELETE /svnadmin/api/v1/groups/{name}", api.deleteGroup)
	mux.HandleFunc("GET /svnadmin/api/v1/groups/{name}/members", api.getGroupMembers)
	mux.HandleFunc("PUT /svnadmin/api/v1/groups/{name}/members", api.setGroupMembers)
	mux.HandleFunc("GET /svnadmin/api/v1/groups/{name}/access-paths", api.getGroupAccessPaths)

	mux.HandleFunc("GET /svnadmin/api/v1/repos", api.listRepositories)
	mux.HandleFunc("POST /svnadmin/api/v1/repos", api.createRepository)
	mux.HandleFunc("GET /svnadmin/api/v1/repos/{repo}/tree", api.getRepositoryTree)
	mux.HandleFunc("GET /svnadmin/api/v1/repos/{repo}/info", api.getRepositoryInfo)
	mux.HandleFunc("DELETE /svnadmin/api/v1/repos/{repo}", api.deleteRepository)

	mux.HandleFunc("GET /svnadmin/api/v1/access-paths", api.listAccessPaths)
	mux.HandleFunc("POST /svnadmin/api/v1/access-paths", api.createAccessPath)
	mux.HandleFunc("DELETE /svnadmin/api/v1/access-paths/{path}", api.deleteAccessPath)
	mux.HandleFunc("GET /svnadmin/api/v1/access-paths/{path}/acl", api.getAccessPathACL)
	mux.HandleFunc("PUT /svnadmin/api/v1/access-paths/{path}/acl/{principal}", api.setAccessPathACL)
	mux.HandleFunc("DELETE /svnadmin/api/v1/access-paths/{path}/acl/{principal}", api.deleteAccessPathACL)
	mux.HandleFunc("GET /svnadmin/api/v1/access-paths/{path}/managers", api.getAccessPathManagers)
	mux.HandleFunc("PUT /svnadmin/api/v1/access-paths/{path}/managers", api.setAccessPathManagers)

	mux.HandleFunc("GET /svnadmin/api/v1/roles", api.listRoles)

	if api.cfg.EnableDocs {
		mux.HandleFunc("GET /svnadmin/api/docs/openapi.json", api.openapiJSON)
		mux.Handle("GET "+docsPrefix, httpSwagger.Handler(
			httpSwagger.URL("openapi.json"),
			httpSwagger.PersistAuthorization(true),
			httpSwagger.DocExpansion("list"),
		))
	} else {
		mux.HandleFunc("GET /svnadmin/api/docs", api.notFound)
		mux.HandleFunc("GET "+docsPrefix, api.notFound)
	}

	// Everything under /svnadmin/api/ that is not explicitly registered stays
	// an honest 501 until P2-P4 implement it. The bare /api/ prefix is also
	// guarded so an API-looking path can never fall through to the SPA.
	mux.HandleFunc(apiPrefix, api.notImplemented)
	mux.HandleFunc("/api/", api.notImplemented)

	// SPA fallback. Unknown non-API paths return index.html (plan D10).
	mux.Handle("/", api.web)

	return api.withRequestID(
		api.accessLog(
			api.recoverer(
				api.securityHeaders(
					api.allowlistMiddleware(
						api.authMiddleware(mux),
					),
				),
			),
		),
	), nil
}
